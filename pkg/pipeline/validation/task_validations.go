package validation

import (
	"fmt"
	"strings"

	"github.com/equinor/radix-tekton/pkg/defaults"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

//ValidateTask Validate task
func ValidateTask(task *v1beta1.Task) []error {
	errs := validateTaskSecretRefDoesNotExist(task)
	errs = append(errs, validateTaskSteps(task)...)
	return errs
}

func validateTaskSteps(task *v1beta1.Task) []error {
	if len(task.Spec.Steps) == 0 {
		return []error{fmt.Errorf("invalid task %s: step list is empty", task.GetName())}
	}
	return nil
}

func validateTaskSecretRefDoesNotExist(task *v1beta1.Task) []error {
	if volumeHasHostPath(task) {
		return errorTaskContainsHostPath(task)
	}
	for _, step := range task.Spec.Steps {
		if containerEnvFromSourceHasNonRadixSecretRef(step.EnvFrom) ||
			containerEnvVarHasNonRadixSecretRef(step.Env) {
			return errorTaskContainsSecretRef(task)
		}
	}
	for _, sidecar := range task.Spec.Sidecars {
		if containerEnvFromSourceHasNonRadixSecretRef(sidecar.EnvFrom) ||
			containerEnvVarHasNonRadixSecretRef(sidecar.Env) {
			return errorTaskContainsSecretRef(task)
		}
	}
	for _, volume := range task.Spec.Volumes {
		if volume.Secret != nil {
			if isRadixBuildSecret(volume.Secret.SecretName) {
				continue
			}
			return errorTaskContainsSecretRef(task)
		}
	}
	if task.Spec.StepTemplate != nil &&
		(containerEnvFromSourceHasNonRadixSecretRef(task.Spec.StepTemplate.EnvFrom) ||
			containerEnvVarHasNonRadixSecretRef(task.Spec.StepTemplate.Env)) {
		return errorTaskContainsSecretRef(task)
	}
	return nil
}

func volumeHasHostPath(task *v1beta1.Task) bool {
	for _, volume := range task.Spec.Volumes {
		if volume.HostPath != nil {
			return true
		}
	}
	return false
}

func errorTaskContainsSecretRef(task *v1beta1.Task) []error {
	return []error{fmt.Errorf("invalid task %s: references to secrets are not allowed", task.GetName())}
}

func errorTaskContainsHostPath(task *v1beta1.Task) []error {
	return []error{fmt.Errorf("invalid task %s: HostPath is not allowed", task.GetName())}
}

func containerEnvFromSourceHasNonRadixSecretRef(envFromSources []corev1.EnvFromSource) bool {
	for _, source := range envFromSources {
		if source.SecretRef != nil {
			return !isRadixBuildSecret(source.SecretRef.Name)
		}
	}
	return false
}

func containerEnvVarHasNonRadixSecretRef(envVars []corev1.EnvVar) bool {
	for _, envVar := range envVars {
		if envVar.ValueFrom != nil && envVar.ValueFrom.SecretKeyRef != nil {
			return !isRadixBuildSecret(envVar.ValueFrom.SecretKeyRef.Name)
		}
	}
	return false
}

func isRadixBuildSecret(secretName string) bool {
	return strings.EqualFold(secretName, defaults.SubstitutionRadixBuildSecretsTarget)
}
