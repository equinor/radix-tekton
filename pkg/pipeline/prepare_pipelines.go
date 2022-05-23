package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/equinor/radix-common/utils"
	commonErrors "github.com/equinor/radix-common/utils/errors"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	"github.com/equinor/radix-tekton/pkg/defaults"
	"github.com/equinor/radix-tekton/pkg/utils/labels"
	"github.com/goccy/go-yaml"
	log "github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (ctx *pipelineContext) preparePipelinesJob() error {
	namespace := ctx.env.GetAppNamespace()
	timestamp := time.Now().Format("20060102150405")
	var errs []error
	for targetEnv := range ctx.targetEnvironments {
		log.Debugf("create a pipeline for the environment %s", targetEnv)
		err := ctx.preparePipelinesJobForTargetEnv(namespace, targetEnv, timestamp)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return commonErrors.Concat(errs)
	}
	return nil
}

func (ctx *pipelineContext) preparePipelinesJobForTargetEnv(namespace, targetEnv, timestamp string) error {
	pipelineFile := defaults.DefaultPipelineFileName //TODO - get pipeline for the targetEnv
	pipelineFilePath, err := ctx.getPipelineFilePath(pipelineFile)
	if err != nil {
		return err
	}

	err = ctx.pipelineFileExists(pipelineFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("There is no Tekton pipeline file: %s. Skip Tekton pipeline", pipelineFilePath)
			return nil
		}
		return err
	}

	pipeline, err := ctx.getPipeline(pipelineFilePath)
	if err != nil {
		return err
	}
	log.Debugf("loaded a pipeline with %d tasks", len(pipeline.Spec.Tasks))

	tasks, err := ctx.getPipelineTasks(pipelineFilePath, pipeline)
	if err != nil {
		return err
	}
	log.Debug("all pipeline tasks found")
	taskMap, err := ctx.createTasks(namespace, targetEnv, tasks, timestamp)
	if err != nil {
		return err
	}

	createdPipeline, err := ctx.createPipeline(namespace, targetEnv, pipeline, taskMap, timestamp)
	if err != nil {
		return err
	}
	log.Infof("created the pipeline %s for the environment %s", createdPipeline.Name, targetEnv)
	return nil
}

func (ctx *pipelineContext) pipelineFileExists(pipelineFilePath string) error {
	_, err := os.Stat(pipelineFilePath)
	return err
}

func (ctx *pipelineContext) createTasks(namespace string, targetEnv string, tasks []v1beta1.Task, timestamp string) (map[string]v1beta1.Task, error) {
	var createTaskErrors []error
	taskMap := make(map[string]v1beta1.Task)
	for _, task := range tasks {
		originalTaskName := task.Name
		createdTask, err := ctx.createTask(namespace, targetEnv, originalTaskName, task, timestamp)
		if err != nil {
			createTaskErrors = append(createTaskErrors, err)
			continue
		}
		taskMap[originalTaskName] = *createdTask
		log.Debugf("created the task %s", task.Name)
	}
	if len(createTaskErrors) > 0 {
		return nil, commonErrors.Concat(createTaskErrors)
	}
	return taskMap, nil
}

func (ctx *pipelineContext) getPipelineTasks(pipelineFilePath string, pipeline *v1beta1.Pipeline) ([]v1beta1.Task, error) {
	taskMap, err := ctx.getTasks(pipelineFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed get tasks: %v", err)
	}
	if len(taskMap) == 0 {
		return nil, fmt.Errorf("no tasks found: %v", err)
	}
	var tasks []v1beta1.Task
	var validateTaskErrors []error
	for _, pipelineSpecTask := range pipeline.Spec.Tasks {
		task, taskExists := taskMap[pipelineSpecTask.TaskRef.Name]
		if !taskExists {
			validateTaskErrors = append(validateTaskErrors, fmt.Errorf("missing task %s", pipelineSpecTask.Name))
			continue
		}
		tasks = append(tasks, task)
	}
	if len(validateTaskErrors) > 0 {
		return nil, commonErrors.Concat(validateTaskErrors)
	}
	return tasks, nil
}

func (ctx *pipelineContext) getPipelineFilePath(pipelineFile string) (string, error) {
	if len(pipelineFile) == 0 {
		pipelineFile = defaults.DefaultPipelineFileName
		log.Debugf("Tekton pipeline file name is not specified, using the default file name %s", defaults.DefaultPipelineFileName)
	}
	if strings.HasPrefix(pipelineFile, "/") {
		pipelineFile = pipelineFile[1:] //Tekton pipeline file path is relative to the root repository folder (also config-folder)
	}
	configFolder := filepath.Dir(ctx.env.GetRadixConfigFileName())
	return filepath.Join(configFolder, pipelineFile), nil
}

func validatePipeline(pipeline *v1beta1.Pipeline) error {
	var validationErrors []error
	if len(pipeline.Spec.Tasks) == 0 {
		validationErrors = append(validationErrors, fmt.Errorf("missing tasks in the pipeline %s", pipeline.Name))
	}
	for i, pipelineSpecTask := range pipeline.Spec.Tasks {
		if len(pipelineSpecTask.Name) == 0 || pipelineSpecTask.TaskRef == nil {
			validationErrors = append(validationErrors,
				fmt.Errorf("invalid task #%d %s: each Task within a Pipeline must have a valid name and a taskRef.\n"+
					"https://tekton.dev/docs/pipelines/pipelines/#adding-tasks-to-the-pipeline",
					i+1, pipelineSpecTask.Name))
		}
	}
	if len(validationErrors) == 0 {
		return nil
	}
	return commonErrors.Concat(validationErrors)
}

func (ctx *pipelineContext) createPipeline(namespace string, targetEnv string, pipeline *v1beta1.Pipeline, taskMap map[string]v1beta1.Task, timestamp string) (*v1beta1.Pipeline, error) {
	originalPipelineName := pipeline.Name
	var setTaskRefErrors []error
	for i, pipelineSpecTask := range pipeline.Spec.Tasks {
		createdTask, ok := taskMap[pipelineSpecTask.TaskRef.Name]
		if !ok {
			setTaskRefErrors = append(setTaskRefErrors, fmt.Errorf("task %s has not been created", pipelineSpecTask.Name))
			continue
		}
		pipeline.Spec.Tasks[i].TaskRef = &v1beta1.TaskRef{Name: createdTask.Name}
	}
	if len(setTaskRefErrors) > 0 {
		return nil, commonErrors.Concat(setTaskRefErrors)
	}
	pipelineName := fmt.Sprintf("radix-pipeline-%s-%s-%s-%s", getShortName(targetEnv), getShortName(originalPipelineName), timestamp, ctx.hash)
	pipeline.ObjectMeta.Name = pipelineName
	pipeline.ObjectMeta.Labels = labels.GetLabelsForEnvironment(ctx, targetEnv)
	pipeline.ObjectMeta.Annotations = map[string]string{
		kube.RadixBranchAnnotation:      ctx.env.GetBranch(),
		defaults.PipelineNameAnnotation: originalPipelineName,
	}
	if ctx.ownerReference != nil {
		pipeline.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*ctx.ownerReference}
	}
	return ctx.tektonClient.TektonV1beta1().Pipelines(namespace).Create(context.Background(), pipeline,
		metav1.CreateOptions{})
}

func (ctx *pipelineContext) createTask(namespace, targetEnv, originalTaskName string, task v1beta1.Task, timestamp string) (*v1beta1.Task, error) {
	taskName := fmt.Sprintf("radix-task-%s-%s-%s-%s", getShortName(targetEnv), getShortName(originalTaskName), timestamp, ctx.hash)
	task.ObjectMeta.Name = taskName
	task.ObjectMeta.Annotations = map[string]string{defaults.PipelineTaskNameAnnotation: originalTaskName}
	task.ObjectMeta.Labels = labels.GetLabelsForEnvironment(ctx, targetEnv)
	if ctx.ownerReference != nil {
		task.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*ctx.ownerReference}
	}
	return ctx.tektonClient.TektonV1beta1().Tasks(namespace).Create(context.Background(), &task,
		metav1.CreateOptions{})
}

func getShortName(name string) string {
	if len(name) > 4 {
		name = name[:4]
	}
	return fmt.Sprintf("%s-%s", name, strings.ToLower(utils.RandStringStrSeed(5, name)))
}

func (ctx *pipelineContext) getPipeline(pipelineFileName string) (*v1beta1.Pipeline, error) {
	pipelineFolder := filepath.Dir(pipelineFileName)
	if _, err := os.Stat(pipelineFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("missing pipeline folder: %s", pipelineFolder)
	}
	pipelineData, err := os.ReadFile(pipelineFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read the pipeline file %s: %v", pipelineFileName, err)
	}
	var pipeline v1beta1.Pipeline
	err = yaml.Unmarshal(pipelineData, &pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to load the pipeline from the file %s: %v", pipelineFileName, err)
	}
	log.Debugf("loaded pipeline %s", pipelineFileName)
	err = validatePipeline(&pipeline)
	if err != nil {
		return nil, err
	}
	return &pipeline, nil
}

func (ctx *pipelineContext) getTasks(pipelineFilePath string) (map[string]v1beta1.Task, error) {
	pipelineFolder := filepath.Dir(pipelineFilePath)
	if _, err := os.Stat(pipelineFolder); os.IsNotExist(err) {
		return nil, fmt.Errorf("missing pipeline folder: %s", pipelineFolder)
	}

	fileNameList, err := filepath.Glob(filepath.Join(pipelineFolder, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to scan pipeline folder %s: %v", pipelineFolder, err)
	}
	fileMap := make(map[interface{}]interface{})
	taskMap := make(map[string]v1beta1.Task)
	for _, fileName := range fileNameList {
		if strings.EqualFold(fileName, pipelineFilePath) {
			continue
		}
		fileData, err := os.ReadFile(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to read the file %s: %v", fileName, err)
		}
		err = yaml.Unmarshal(fileData, &fileMap)
		if err != nil {
			return nil, fmt.Errorf("failed to read data from the file %s: %v", fileName, err)
		}
		if !fileMapContainsTektonTask(fileMap) {
			log.Debugf("skip the file %s - not a Tekton task", fileName)
			continue
		}
		var task v1beta1.Task
		err = yaml.Unmarshal(fileData, &task)
		if err != nil {
			return nil, fmt.Errorf("failed to load the task from the file %s: %v", fileData, err)
		}
		taskMap[task.Name] = task
	}
	return taskMap, nil
}

func fileMapContainsTektonTask(fileMap map[interface{}]interface{}) bool {
	if kind, hasKind := fileMap["kind"]; !hasKind || fmt.Sprintf("%v", kind) != "Task" {
		return false
	}
	apiVersion, hasApiVersion := fileMap["apiVersion"]
	return hasApiVersion && strings.HasPrefix(fmt.Sprintf("%v", apiVersion), "tekton.dev/")
}
