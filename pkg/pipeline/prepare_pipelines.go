package pipeline

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/equinor/radix-common/utils"
	commonUtils "github.com/equinor/radix-common/utils"
	commonErrors "github.com/equinor/radix-common/utils/errors"
	"github.com/equinor/radix-common/utils/maps"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	"github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-tekton/pkg/defaults"
	"github.com/equinor/radix-tekton/pkg/pipeline/validation"
	"github.com/equinor/radix-tekton/pkg/utils/configmap"
	"github.com/equinor/radix-tekton/pkg/utils/git"
	"github.com/equinor/radix-tekton/pkg/utils/labels"
	"github.com/equinor/radix-tekton/pkg/utils/radix/deployment/commithash"
	"github.com/goccy/go-yaml"
	log "github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type componentsInEnvironments map[string][]string //map[envName]componentNames

func (ctx *pipelineContext) preparePipelinesJob() (componentsInEnvironments, bool, error) {
	namespace := ctx.env.GetAppNamespace()
	timestamp := time.Now().Format("20060102150405")
	componentsChangedInEnv := make(componentsInEnvironments)
	radixConfigWasChanged := false

	if ctx.env.GetRadixPipelineType() == v1.BuildDeploy {
		changedComponents, changedRadixConfig, err := ctx.prepareBuildDeployPipeline()
		if err != nil {
			return nil, false, err
		}
		componentsChangedInEnv = changedComponents
		radixConfigWasChanged = changedRadixConfig
	}

	var errs []error
	for targetEnv := range ctx.targetEnvironments {
		log.Debugf("create a pipeline for the environment %s", targetEnv)
		err := ctx.preparePipelinesJobForTargetEnv(namespace, targetEnv, timestamp)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return componentsChangedInEnv, radixConfigWasChanged, commonErrors.Concat(errs)
}

func (ctx *pipelineContext) prepareBuildDeployPipeline() (componentsInEnvironments, bool, error) {
	targetEnvs := maps.GetKeysFromMap(ctx.targetEnvironments)
	env := ctx.GetEnv()
	pipelineTargetCommitHash, commitTags, err := git.GetCommitHashAndTags(env)
	if err != nil {
		return nil, false, err
	}
	err = configmap.CreateGitConfigFromGitRepository(env, ctx.kubeClient, pipelineTargetCommitHash, commitTags)
	if err != nil {
		return nil, false, err
	}

	radixDeploymentCommitHashProvider := commithash.NewProvider(ctx.radixClient, env.GetAppName(), targetEnvs)
	lastCommitHashesForEnvs, err := radixDeploymentCommitHashProvider.GetLastCommitHashesForEnvironments()
	if err != nil {
		return nil, false, err
	}

	changesFromGitRepository, radixConfigWasChanged, err := git.GetChangesFromGitRepository(env.GetGitRepositoryWorkspace(), env.GetRadixConfigBranch(), env.GetRadixConfigFileName(), pipelineTargetCommitHash, lastCommitHashesForEnvs)
	if err != nil {
		return nil, false, err
	}

	componentsChangedInEnv := componentsInEnvironments{}
	for _, radixComponent := range ctx.GetRadixApplication().Spec.Components {
		if len(radixComponent.Image) > 0 {
			continue
		}
		for envName, changedFolders := range changesFromGitRepository {
			environmentConfig := radixComponent.GetEnvironmentConfigByName(envName)
			if !radixComponent.GetEnabledForEnv(environmentConfig) {
				continue
			}
			sourceFolder := commonUtils.TernaryString(len(radixComponent.GetSourceFolder()) == 0, ".", radixComponent.GetSourceFolder())
			sourceFolderWithTrailingSlash := sourceFolder
			if !strings.HasSuffix(sourceFolderWithTrailingSlash, "/") {
				sourceFolderWithTrailingSlash = path.Join(sourceFolderWithTrailingSlash, "/")
			}
			for _, folder := range changedFolders {
				if strings.HasPrefix(folder, sourceFolderWithTrailingSlash) {
					componentsChangedInEnv[envName] = append(componentsChangedInEnv[envName], radixComponent.GetName())
					break
				}
			}
		}
	}
	return componentsChangedInEnv, radixConfigWasChanged, nil
}

func (ctx *pipelineContext) preparePipelinesJobForTargetEnv(namespace, targetEnv, timestamp string) error {
	pipelineFilePath, err := ctx.getPipelineFilePath("") //TODO - get pipeline for the targetEnv
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
		ensureCorrectSecureContext(&task)
		createdTask, err := ctx.createTask(namespace, targetEnv, originalTaskName, task, timestamp)
		if err != nil {
			createTaskErrors = append(createTaskErrors, err)
			continue
		}
		taskMap[originalTaskName] = *createdTask
		log.Debugf("created the task %s", task.Name)
	}
	return taskMap, commonErrors.Concat(createTaskErrors)
}

func ensureCorrectSecureContext(task *v1beta1.Task) {
	for _, step := range task.Spec.Steps {
		setNotElevatedPrivileges(step.SecurityContext)
	}
	for _, sidecar := range task.Spec.Sidecars {
		setNotElevatedPrivileges(sidecar.SecurityContext)
	}
	if task.Spec.StepTemplate != nil {
		setNotElevatedPrivileges(task.Spec.StepTemplate.SecurityContext)
	}
}

func setNotElevatedPrivileges(securityContext *corev1.SecurityContext) {
	if securityContext == nil {
		return
	}
	securityContext.RunAsNonRoot = commonUtils.BoolPtr(true)
	securityContext.Privileged = commonUtils.BoolPtr(false)
	securityContext.AllowPrivilegeEscalation = commonUtils.BoolPtr(false)
	securityContext.Capabilities.Drop = []corev1.Capability{"ALL"}
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
			validateTaskErrors = append(validateTaskErrors, fmt.Errorf("missing the pipeline task %s, referenced to the task %s", pipelineSpecTask.Name, pipelineSpecTask.TaskRef.Name))
			continue
		}
		validateTaskErrors = append(validateTaskErrors, validation.ValidateTask(&task)...)
		tasks = append(tasks, task)
	}
	return tasks, commonErrors.Concat(validateTaskErrors)
}

func (ctx *pipelineContext) getPipelineFilePath(pipelineFile string) (string, error) {
	if len(pipelineFile) == 0 {
		pipelineFile = defaults.DefaultPipelineFileName
		log.Debugf("Tekton pipeline file name is not specified, using the default file name %s", defaults.DefaultPipelineFileName)
	}
	pipelineFile = strings.TrimPrefix(pipelineFile, "/") //Tekton pipeline folder currently is relative to the Radix config file repository folder
	configFolder := filepath.Dir(ctx.env.GetRadixConfigFileName())
	return filepath.Join(configFolder, pipelineFile), nil
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
	err = validation.ValidatePipeline(&pipeline)
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
		fileData = []byte(strings.ReplaceAll(string(fileData), defaults.SubstitutionRadixBuildSecretsSource, defaults.SubstitutionRadixBuildSecretsTarget))
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
