package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	commonErrors "github.com/equinor/radix-common/utils/errors"
	operatorDefaults "github.com/equinor/radix-operator/pkg/apis/defaults"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	radixv1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils"
	"github.com/equinor/radix-tekton/pkg/defaults"
	"github.com/equinor/radix-tekton/pkg/utils/labels"
	"github.com/equinor/radix-tekton/pkg/utils/radix/applicationconfig"
	log "github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RunPipelinesJob Run the job, which creates Tekton PipelineRun-s for each preliminary prepared pipelines of the specified branch
func (ctx *pipelineContext) RunPipelinesJob() error {
	if ctx.GetEnv().GetRadixPipelineType() == radixv1.Build {
		log.Infof("pipeline type is build, skip Tekton pipeline run.")
		return nil
	}
	namespace := ctx.env.GetAppNamespace()
	pipelineList, err := ctx.tektonClient.TektonV1().Pipelines(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", kube.RadixJobNameLabel, ctx.env.GetRadixPipelineJobName()),
	})
	if err != nil {
		return err
	}
	if len(pipelineList.Items) == 0 {
		log.Infof("no pipelines exist, skip Tekton pipeline run.")
		return nil
	}

	radixApplication, err := ctx.createRadixApplicationFromConfigMap()
	if err != nil {
		return err
	}
	ctx.radixApplication = radixApplication
	err = ctx.setTargetEnvironments()
	if err != nil {
		return err
	}

	tektonPipelineBranch := ctx.env.GetBranch()
	if ctx.GetEnv().GetRadixPipelineType() == radixv1.Deploy {
		re := applicationconfig.GetEnvironmentFromRadixApplication(ctx.radixApplication, ctx.env.GetRadixDeployToEnvironment())
		tektonPipelineBranch = re.Build.From
	}
	log.Infof("Run tekton pipelines for the branch %s", tektonPipelineBranch)

	pipelineRunMap, err := ctx.runPipelines(pipelineList.Items, namespace)

	if err != nil {
		return fmt.Errorf("failed to run pipelines: %w", err)
	}

	err = ctx.GetPipelineRunsWaiter().Wait(pipelineRunMap, ctx.env)
	if err != nil {
		return fmt.Errorf("failed tekton pipelines for the application %s, for environment(s) %s. %w",
			ctx.env.GetAppName(),
			ctx.getTargetEnvsAsString(),
			err)
	}
	return nil
}

func (ctx *pipelineContext) getTargetEnvsAsString() string {
	var envs []string
	for envName := range ctx.targetEnvironments {
		envs = append(envs, envName)
	}
	return strings.Join(envs, ", ")
}

func (ctx *pipelineContext) runPipelines(pipelines []pipelinev1.Pipeline, namespace string) (map[string]*pipelinev1.PipelineRun, error) {
	timestamp := time.Now().Format("20060102150405")
	pipelineRunMap := make(map[string]*pipelinev1.PipelineRun)
	var errs []error
	for _, pipeline := range pipelines {
		createdPipelineRun, err := ctx.createPipelineRun(namespace, &pipeline, timestamp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		pipelineRunMap[createdPipelineRun.GetName()] = createdPipelineRun
	}
	return pipelineRunMap, commonErrors.Concat(errs)
}

func (ctx *pipelineContext) createPipelineRun(namespace string, pipeline *pipelinev1.Pipeline, timestamp string) (*pipelinev1.PipelineRun, error) {
	targetEnv, pipelineTargetEnvDefined := pipeline.ObjectMeta.Labels[kube.RadixEnvLabel]
	if !pipelineTargetEnvDefined {
		return nil, fmt.Errorf("missing target environment in labels of the pipeline %s", pipeline.Name)
	}

	log.Debugf("run pipelinerun for the target environment %s", targetEnv)
	if _, ok := ctx.targetEnvironments[targetEnv]; !ok {
		return nil, fmt.Errorf("missing target environment %s for the pipeline %s", targetEnv, pipeline.Name)
	}

	pipelineRun := ctx.buildPipelineRun(pipeline, targetEnv, timestamp)

	return ctx.tektonClient.TektonV1().PipelineRuns(namespace).Create(context.Background(), &pipelineRun, metav1.CreateOptions{})
}

func (ctx *pipelineContext) buildPipelineRun(pipeline *pipelinev1.Pipeline, targetEnv, timestamp string) pipelinev1.PipelineRun {
	originalPipelineName := pipeline.ObjectMeta.Annotations[defaults.PipelineNameAnnotation]
	pipelineRunName := fmt.Sprintf("radix-pipelinerun-%s-%s-%s", getShortName(targetEnv), timestamp, ctx.hash)
	pipelineParams := ctx.getPipelineParams(pipeline, targetEnv)
	pipelineRun := pipelinev1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:   pipelineRunName,
			Labels: labels.GetLabelsForEnvironment(ctx, targetEnv),
			Annotations: map[string]string{
				kube.RadixBranchAnnotation:      ctx.env.GetBranch(),
				defaults.PipelineNameAnnotation: originalPipelineName,
			},
		},
		Spec: pipelinev1.PipelineRunSpec{
			PipelineRef: &pipelinev1.PipelineRef{Name: pipeline.GetName()},
			Params:      pipelineParams,
			TaskRunTemplate: pipelinev1.PipelineTaskRunTemplate{
				PodTemplate:        ctx.buildPipelineRunPodTemplate(),
				ServiceAccountName: utils.GetSubPipelineServiceAccountName(targetEnv),
			},
		},
	}
	if ctx.ownerReference != nil {
		pipelineRun.ObjectMeta.OwnerReferences = []metav1.OwnerReference{*ctx.ownerReference}
	}
	var taskRunSpecs []pipelinev1.PipelineTaskRunSpec
	for _, task := range pipeline.Spec.Tasks {
		taskRunSpecs = append(taskRunSpecs, pipelineRun.GetTaskRunSpec(task.Name))
	}
	pipelineRun.Spec.TaskRunSpecs = taskRunSpecs
	return pipelineRun
}

func (ctx *pipelineContext) buildPipelineRunPodTemplate() *pod.Template {
	podTemplate := pod.Template{
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot: utils.BoolPtr(true),
		},
	}

	if ctx.radixApplication != nil && len(ctx.radixApplication.Spec.PrivateImageHubs) > 0 {
		podTemplate.ImagePullSecrets = []corev1.LocalObjectReference{{Name: operatorDefaults.PrivateImageHubSecretName}}
	}

	return &podTemplate
}

func (ctx *pipelineContext) getPipelineParams(pipeline *pipelinev1.Pipeline, targetEnv string) []pipelinev1.Param {
	envVars := ctx.getEnvVars(targetEnv)
	pipelineParamsMap := getPipelineParamSpecsMap(pipeline)
	var pipelineParams []pipelinev1.Param
	for envVarName, envVarValue := range envVars {
		paramSpec, envVarExistInParamSpecs := pipelineParamsMap[envVarName]
		if !envVarExistInParamSpecs {
			continue // Add to pipelineRun params only env-vars, existing in the pipeline paramSpecs
		}
		param := pipelinev1.Param{
			Name: envVarName,
			Value: pipelinev1.ParamValue{
				Type: paramSpec.Type,
			},
		}
		if param.Value.Type == pipelinev1.ParamTypeArray { // Param can contain a string value or a comma-separated values array
			param.Value.ArrayVal = strings.Split(envVarValue, ",")
		} else {
			param.Value.StringVal = envVarValue
		}
		pipelineParams = append(pipelineParams, param)
	}
	return pipelineParams
}

func getPipelineParamSpecsMap(pipeline *pipelinev1.Pipeline) map[string]pipelinev1.ParamSpec {
	paramSpecMap := make(map[string]pipelinev1.ParamSpec)
	for _, paramSpec := range pipeline.PipelineSpec().Params {
		paramSpecMap[paramSpec.Name] = paramSpec
	}
	return paramSpecMap
}
