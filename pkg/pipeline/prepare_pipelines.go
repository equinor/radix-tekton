package pipeline

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	commonUtils "github.com/equinor/radix-common/utils"
	commonErrors "github.com/equinor/radix-common/utils/errors"
	"github.com/equinor/radix-common/utils/maps"
	"github.com/equinor/radix-operator/pipeline-runner/model"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
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

func (ctx *pipelineContext) preparePipelinesJob() (*model.PrepareBuildContext, error) {
	buildContext := model.PrepareBuildContext{}

	gitHash, err := ctx.getGitHash()
	if err != nil {
		return nil, err
	}
	if gitHash == "" && ctx.env.GetRadixPipelineType() != v1.BuildDeploy {
		// if no git hash, don't run sub-pipelines
		return &buildContext, nil
	}

	err = git.ResetGitHead(ctx.env.GetGitRepositoryWorkspace(), gitHash)
	if err != nil {
		return nil, err
	}

	if ctx.env.GetRadixPipelineType() == v1.BuildDeploy {
		changedComponents, changedRadixConfig, err := ctx.prepareBuildDeployPipeline()
		if err != nil {
			return nil, err
		}
		buildContext.EnvironmentsToBuild = changedComponents
		buildContext.ChangedRadixConfig = changedRadixConfig
	}

	environmentSubPipelinesToRun, err := ctx.getEnvironmentSubPipelinesToRun()
	if err != nil {
		return nil, err
	}
	buildContext.EnvironmentSubPipelinesToRun = environmentSubPipelinesToRun
	return &buildContext, err
}

func (ctx *pipelineContext) getEnvironmentSubPipelinesToRun() ([]model.EnvironmentSubPipelineToRun, error) {
	var environmentSubPipelinesToRun []model.EnvironmentSubPipelineToRun
	var errs []error
	timestamp := time.Now().Format("20060102150405")
	for targetEnv := range ctx.targetEnvironments {
		log.Debugf("create a sub-pipeline for the environment %s", targetEnv)
		runSubPipeline, pipelineFilePath, err := ctx.preparePipelinesJobForTargetEnv(targetEnv, timestamp)
		if err != nil {
			errs = append(errs, err)
		}
		if runSubPipeline {
			environmentSubPipelinesToRun = append(environmentSubPipelinesToRun, model.EnvironmentSubPipelineToRun{
				Environment:  targetEnv,
				PipelineFile: pipelineFilePath,
			})
		}
	}
	err := commonErrors.Concat(errs)
	if err != nil {
		return nil, err
	}
	if len(environmentSubPipelinesToRun) > 0 {
		log.Infoln("Run sub-pipelines:")
		for _, subPipelineToRun := range environmentSubPipelinesToRun {
			log.Infof("- environment %s, pipeline file %s", subPipelineToRun.Environment, subPipelineToRun.PipelineFile)
		}
		return environmentSubPipelinesToRun, nil
	}
	log.Infoln("No sub-pipelines to run")
	return nil, nil
}

func (ctx *pipelineContext) prepareBuildDeployPipeline() ([]model.EnvironmentToBuild, bool, error) {
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

	if !ctx.isPipelineStartedByWebhook() {
		return nil, false, err
	}

	radixDeploymentCommitHashProvider := commithash.NewProvider(ctx.kubeClient, ctx.radixClient, env.GetAppName(), targetEnvs)
	lastCommitHashesForEnvs, err := radixDeploymentCommitHashProvider.GetLastCommitHashesForEnvironments()
	if err != nil {
		return nil, false, err
	}

	changesFromGitRepository, radixConfigWasChanged, err := git.GetChangesFromGitRepository(env.GetGitRepositoryWorkspace(), env.GetRadixConfigBranch(), env.GetRadixConfigFileName(), pipelineTargetCommitHash, lastCommitHashesForEnvs)
	if err != nil {
		return nil, false, err
	}

	environmentsToBuild := ctx.getEnvironmentsToBuild(changesFromGitRepository)
	return environmentsToBuild, radixConfigWasChanged, nil
}

func (ctx *pipelineContext) isPipelineStartedByWebhook() bool {
	return len(ctx.GetEnv().GetWebhookCommitId()) > 0
}

func (ctx *pipelineContext) getEnvironmentsToBuild(changesFromGitRepository map[string][]string) []model.EnvironmentToBuild {
	var environmentsToBuild []model.EnvironmentToBuild
	for envName, changedFolders := range changesFromGitRepository {
		var componentsWithChangedSource []string
		for _, radixComponent := range ctx.GetRadixApplication().Spec.Components {
			if componentHasChangedSource(envName, &radixComponent, changedFolders) {
				componentsWithChangedSource = append(componentsWithChangedSource, radixComponent.GetName())
			}
		}
		for _, radixJobComponent := range ctx.GetRadixApplication().Spec.Jobs {
			if componentHasChangedSource(envName, &radixJobComponent, changedFolders) {
				componentsWithChangedSource = append(componentsWithChangedSource, radixJobComponent.GetName())
			}
		}
		environmentsToBuild = append(environmentsToBuild, model.EnvironmentToBuild{
			Environment: envName,
			Components:  componentsWithChangedSource,
		})
	}
	return environmentsToBuild
}

func componentHasChangedSource(envName string, component v1.RadixCommonComponent, changedFolders []string) bool {
	if len(component.GetImage()) > 0 {
		return false
	}
	environmentConfig := component.GetEnvironmentConfigByName(envName)
	if !component.GetEnabledForEnv(environmentConfig) {
		return false
	}

	sourceFolder := cleanPathAndSurroundBySlashes(component.GetSourceFolder())
	if path.Dir(sourceFolder) == path.Dir("/") && len(changedFolders) > 0 {
		return true // for components with the repository root as a 'src' - changes in any repository sub-folders are considered also as the component changes
	}

	for _, folder := range changedFolders {
		if strings.HasPrefix(cleanPathAndSurroundBySlashes(folder), sourceFolder) {
			return true
		}
	}
	return false
}

func cleanPathAndSurroundBySlashes(dir string) string {
	if !strings.HasSuffix(dir, "/") {
		dir = fmt.Sprintf("%s/", dir)
	}
	dir = fmt.Sprintf("%s/", path.Dir(dir))
	if !strings.HasPrefix(dir, "/") {
		return fmt.Sprintf("/%s", dir)
	}
	return dir
}

func (ctx *pipelineContext) preparePipelinesJobForTargetEnv(envName, timestamp string) (bool, string, error) {
	pipelineFilePath, err := ctx.getPipelineFilePath("") // TODO - get pipeline for the envName
	if err != nil {
		return false, "", err
	}

	err = ctx.pipelineFileExists(pipelineFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("There is no Tekton pipeline file: %s. Skip Tekton pipeline", pipelineFilePath)
			return false, "", nil
		}
		return false, "", err
	}

	pipeline, err := getPipeline(pipelineFilePath)
	if err != nil {
		return false, "", err
	}
	log.Debugf("loaded a pipeline with %d tasks", len(pipeline.Spec.Tasks))

	tasks, err := ctx.getPipelineTasks(pipelineFilePath, pipeline)
	if err != nil {
		return false, "", err
	}
	log.Debug("all pipeline tasks found")
	err = ctx.createPipeline(envName, pipeline, tasks, timestamp)
	if err != nil {
		return false, "", err
	}
	return true, pipelineFilePath, nil
}

func (ctx *pipelineContext) pipelineFileExists(pipelineFilePath string) error {
	_, err := os.Stat(pipelineFilePath)
	return err
}

func (ctx *pipelineContext) buildTasks(envName string, tasks []v1beta1.Task, timestamp string) map[string]v1beta1.Task {
	taskMap := make(map[string]v1beta1.Task)
	for _, task := range tasks {
		originalTaskName := task.Name
		taskName := fmt.Sprintf("radix-task-%s-%s-%s-%s", getShortName(envName), getShortName(originalTaskName), timestamp, ctx.hash)
		task.ObjectMeta.Name = taskName
		task.ObjectMeta.Annotations = map[string]string{defaults.PipelineTaskNameAnnotation: originalTaskName}
		task.ObjectMeta.Labels = labels.GetLabelsForEnvironment(ctx, envName)
		if ctx.ownerReference != nil {
			task.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*ctx.ownerReference}
		}
		ensureCorrectSecureContext(&task)
		taskMap[originalTaskName] = task
		log.Debugf("created the task %s", task.Name)
	}
	return taskMap
}

func ensureCorrectSecureContext(task *v1beta1.Task) {
	for _, step := range task.Spec.Steps {
		if step.SecurityContext == nil {
			step.SecurityContext = &corev1.SecurityContext{}
		}
		setNotElevatedPrivileges(step.SecurityContext)
	}
	for _, sidecar := range task.Spec.Sidecars {
		if sidecar.SecurityContext == nil {
			sidecar.SecurityContext = &corev1.SecurityContext{}
		}
		setNotElevatedPrivileges(sidecar.SecurityContext)
	}
	if task.Spec.StepTemplate != nil {
		if task.Spec.StepTemplate.SecurityContext == nil {
			task.Spec.StepTemplate.SecurityContext = &corev1.SecurityContext{}
		}
		setNotElevatedPrivileges(task.Spec.StepTemplate.SecurityContext)
	}
}

func setNotElevatedPrivileges(securityContext *corev1.SecurityContext) {
	securityContext.RunAsNonRoot = commonUtils.BoolPtr(true)
	securityContext.Privileged = commonUtils.BoolPtr(false)
	securityContext.AllowPrivilegeEscalation = commonUtils.BoolPtr(false)
	if securityContext.Capabilities == nil {
		securityContext.Capabilities = &corev1.Capabilities{}
	}
	securityContext.Capabilities.Drop = []corev1.Capability{"ALL"}
}

func (ctx *pipelineContext) getPipelineTasks(pipelineFilePath string, pipeline *v1beta1.Pipeline) ([]v1beta1.Task, error) {
	taskMap, err := getTasks(pipelineFilePath)
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
	pipelineFile = strings.TrimPrefix(pipelineFile, "/") // Tekton pipeline folder currently is relative to the Radix config file repository folder
	configFolder := filepath.Dir(ctx.env.GetRadixConfigFileName())
	return filepath.Join(configFolder, pipelineFile), nil
}

func (ctx *pipelineContext) createPipeline(envName string, pipeline *v1beta1.Pipeline, tasks []v1beta1.Task, timestamp string) error {
	taskMap := ctx.buildTasks(envName, tasks, timestamp)

	originalPipelineName := pipeline.Name
	var errs []error
	for i, pipelineSpecTask := range pipeline.Spec.Tasks {
		task, ok := taskMap[pipelineSpecTask.TaskRef.Name]
		if !ok {
			errs = append(errs, fmt.Errorf("task %s has not been created", pipelineSpecTask.Name))
			continue
		}
		pipeline.Spec.Tasks[i].TaskRef = &v1beta1.TaskRef{Name: task.Name}
	}
	if len(errs) > 0 {
		return commonErrors.Concat(errs)
	}
	pipelineName := fmt.Sprintf("radix-pipeline-%s-%s-%s-%s", getShortName(envName), getShortName(originalPipelineName), timestamp, ctx.hash)
	pipeline.ObjectMeta.Name = pipelineName
	pipeline.ObjectMeta.Labels = labels.GetLabelsForEnvironment(ctx, envName)
	pipeline.ObjectMeta.Annotations = map[string]string{
		kube.RadixBranchAnnotation:      ctx.env.GetBranch(),
		defaults.PipelineNameAnnotation: originalPipelineName,
	}
	if ctx.ownerReference != nil {
		pipeline.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*ctx.ownerReference}
	}
	err := ctx.createTasks(taskMap)
	if err != nil {
		return fmt.Errorf("tasks have not been created. Error: %w", err)
	}
	log.Infof("creates %d tasks for the environment %s", len(taskMap), envName)

	_, err = ctx.tektonClient.TektonV1beta1().Pipelines(ctx.env.GetAppNamespace()).Create(context.Background(), pipeline, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("pipeline %s has not been created. Error: %w", pipeline.Name, err)
	}
	log.Infof("created the pipeline %s for the environment %s", pipeline.Name, envName)
	return nil
}

func (ctx *pipelineContext) createTasks(taskMap map[string]v1beta1.Task) error {
	namespace := ctx.env.GetAppNamespace()
	var errs []error
	for _, task := range taskMap {
		_, err := ctx.tektonClient.TektonV1beta1().Tasks(namespace).Create(context.Background(), &task,
			metav1.CreateOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("task %s has not been created. Error: %w", task.Name, err))
		}
	}
	return commonErrors.Concat(errs)
}

func getShortName(name string) string {
	if len(name) > 4 {
		name = name[:4]
	}
	return fmt.Sprintf("%s-%s", name, strings.ToLower(commonUtils.RandStringStrSeed(5, name)))
}

func getPipeline(pipelineFileName string) (*v1beta1.Pipeline, error) {
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

func getTasks(pipelineFilePath string) (map[string]v1beta1.Task, error) {
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
