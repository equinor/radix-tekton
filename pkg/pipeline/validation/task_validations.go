package validation

import (
	errors2 "errors"
	"strings"

	operatorDefaults "github.com/equinor/radix-operator/pkg/apis/defaults"
	"github.com/equinor/radix-tekton/pkg/defaults"
	"github.com/pkg/errors"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	ErrEmptyStepList             = errors.New("step list is empty")
	ErrSecretReferenceNotAllowed = errors.New("references to secrets are not allowed")
	ErrRadixVolumeNameNotAllowed = errors.New("volume name starting with radix are not allowed")
	ErrHostPathNotAllowed        = errors.New("HostPath is not allowed")
)

// ValidateTask Validate task
func ValidateTask(task *pipelinev1.Task) error {
	var errs []error
	errs = append(errs, validateTaskSecretRefDoesNotExist(task)...)
	errs = append(errs, validateVolumeName(task)...)
	errs = append(errs, validateTaskSteps(task)...)
	err := errors2.Join(errs...)

	if err != nil {
		return errors.Wrapf(err, "Task %s is invalid", task.GetName())
	}

	return nil
}

func validateTaskSteps(task *pipelinev1.Task) []error {
	if len(task.Spec.Steps) == 0 {
		return []error{ErrEmptyStepList}
	}
	return nil
}

func validateVolumeName(task *pipelinev1.Task) []error {
	var errs []error

	for _, volume := range task.Spec.Volumes {

		if !strings.HasPrefix(volume.Name, "radix") {
			continue
		}

		if volume.Secret != nil && (isRadixBuildSecret(volume.Secret.SecretName) ||
			isRadixGitDeployKeySecret(volume.Secret.SecretName)) {
			continue
		}

		errs = append(errs, errorTaskContainsInvalidVolumeName(volume))
	}

	return errs
}

func validateTaskSecretRefDoesNotExist(task *pipelinev1.Task) []error {
	var errs []error

	if volumeHasHostPath(task) {
		errs = append(errs, ErrHostPathNotAllowed)
	}
	for _, step := range task.Spec.Steps {
		if containerEnvFromSourceHasNonRadixSecretRef(step.EnvFrom) ||
			containerEnvVarHasNonRadixSecretRef(step.Env) {

			errs = append(errs, ErrSecretReferenceNotAllowed)
		}
	}
	for _, sidecar := range task.Spec.Sidecars {
		if containerEnvFromSourceHasNonRadixSecretRef(sidecar.EnvFrom) ||
			containerEnvVarHasNonRadixSecretRef(sidecar.Env) {

			errs = append(errs, ErrSecretReferenceNotAllowed)
		}
	}
	for _, volume := range task.Spec.Volumes {
		if volume.Secret != nil {
			if isRadixBuildSecret(volume.Secret.SecretName) ||
				isRadixGitDeployKeySecret(volume.Secret.SecretName) {
				continue
			}

			errs = append(errs, ErrSecretReferenceNotAllowed)
		}
	}
	if task.Spec.StepTemplate != nil &&
		(containerEnvFromSourceHasNonRadixSecretRef(task.Spec.StepTemplate.EnvFrom) ||
			containerEnvVarHasNonRadixSecretRef(task.Spec.StepTemplate.Env)) {

		errs = append(errs, ErrSecretReferenceNotAllowed)
	}
	return errs
}

func volumeHasHostPath(task *pipelinev1.Task) bool {
	for _, volume := range task.Spec.Volumes {
		if volume.HostPath != nil {
			return true
		}
	}
	return false
}

func errorTaskContainsInvalidVolumeName(volume corev1.Volume) error {
	return errors.WithMessagef(ErrRadixVolumeNameNotAllowed, "volume %s has invalid name", volume.Name)
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
func isRadixGitDeployKeySecret(secretName string) bool {
	return strings.EqualFold(secretName, operatorDefaults.GitPrivateKeySecretName)
}
