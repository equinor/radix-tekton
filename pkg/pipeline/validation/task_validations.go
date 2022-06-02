package validation

import (
	"fmt"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

//ValidateTask Validate task
func ValidateTask(task *v1beta1.Task) []error {
	return validateTaskSecretRefDoesNotExist(task)
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
