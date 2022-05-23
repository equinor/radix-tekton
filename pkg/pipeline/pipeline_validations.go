package pipeline

import (
	"fmt"
	commonErrors "github.com/equinor/radix-common/utils/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

func validatePipeline(pipeline *v1beta1.Pipeline) error {
	var validationErrors []error

	validationErrors = append(validationErrors, validatePipelineTasks(pipeline)...)

	if len(validationErrors) == 0 {
		return nil
	}
	return commonErrors.Concat(validationErrors)
}

func validatePipelineTasks(pipeline *v1beta1.Pipeline) []error {
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
	return validationErrors
}

func validateTaskSecretRefDoesNotExist(task *v1beta1.Task) []error {
	for _, step := range task.Spec.Steps {
		if containerEnvFromSourceHasSecretRef(step.Container.EnvFrom) ||
			containerEnvVarHasSecretRef(step.Container.Env) {
			return errorTaskContainsSecretRef(task)
		}
	}
	for _, sidecar := range task.Spec.Sidecars {
		if containerEnvFromSourceHasSecretRef(sidecar.Container.EnvFrom) ||
			containerEnvVarHasSecretRef(sidecar.Container.Env) {
			return errorTaskContainsSecretRef(task)
		}
	}
	for _, volume := range task.Spec.Volumes {
		if volume.Secret != nil { //TBD - probably some secrets can be allowed
			return errorTaskContainsSecretRef(task)
		}
	}
	if task.Spec.StepTemplate != nil &&
		(containerEnvFromSourceHasSecretRef(task.Spec.StepTemplate.EnvFrom) ||
			containerEnvVarHasSecretRef(task.Spec.StepTemplate.Env)) {
		return errorTaskContainsSecretRef(task)
	}
	return nil
}

func errorTaskContainsSecretRef(task *v1beta1.Task) []error {
	return []error{fmt.Errorf("invalid task %s: references to secrets are not allowed", task.GetName())}
}

func containerEnvFromSourceHasSecretRef(envFromSources []corev1.EnvFromSource) bool {
	for _, source := range envFromSources {
		if source.SecretRef != nil {
			return true
		}
	}
	return false
}

func containerEnvVarHasSecretRef(envVars []corev1.EnvVar) bool {
	for _, envVar := range envVars {
		if envVar.ValueFrom != nil && envVar.ValueFrom.SecretKeyRef != nil {
			return true
		}
	}
	return false
}
