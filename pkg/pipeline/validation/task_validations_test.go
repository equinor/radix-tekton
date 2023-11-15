package validation_test

import (
	"testing"

	operatorDefaults "github.com/equinor/radix-operator/pkg/apis/defaults"
	"github.com/equinor/radix-tekton/pkg/defaults"
	"github.com/equinor/radix-tekton/pkg/pipeline/validation"
	"github.com/stretchr/testify/assert"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInvalidTask(t *testing.T) {
	errs := validation.ValidateTask(&pipelinev1.Task{
		ObjectMeta: v1.ObjectMeta{Name: "Test Task"},
		Spec:       pipelinev1.TaskSpec{},
	})

	assert.ErrorIs(t, errs, validation.ErrEmptyStepList)
}

func TestValidTask(t *testing.T) {
	errs := validation.ValidateTask(&pipelinev1.Task{
		ObjectMeta: v1.ObjectMeta{Name: "Test Task"},
		Spec: pipelinev1.TaskSpec{
			Steps: []pipelinev1.Step{pipelinev1.Step{}},
		},
	})

	assert.Empty(t, errs)
}

func TestNoSecretsAllowed(t *testing.T) {
	err := validation.ValidateTask(&pipelinev1.Task{
		ObjectMeta: v1.ObjectMeta{Name: "Test Task"},
		Spec: pipelinev1.TaskSpec{
			Steps: []pipelinev1.Step{pipelinev1.Step{}},
			Volumes: []corev1.Volume{corev1.Volume{
				Name: "testing-secret-mount",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "test-illegal-secret",
					},
				},
			}},
		},
	})

	assert.ErrorIs(t, err, validation.ErrSecretReferenceNotAllowed)
}

func TestNoRadixVolumeAllowed(t *testing.T) {
	err := validation.ValidateTask(&pipelinev1.Task{
		ObjectMeta: v1.ObjectMeta{Name: "Test Task"},
		Spec: pipelinev1.TaskSpec{
			Steps: []pipelinev1.Step{pipelinev1.Step{}},
			Volumes: []corev1.Volume{corev1.Volume{
				Name: "radix-hello-world",
			}},
		},
	})

	assert.ErrorIs(t, err, validation.ErrRadixVolumeNameNotAllowed)
}

func TestSpecialRadixVolumeAllowed(t *testing.T) {
	err := validation.ValidateTask(&pipelinev1.Task{
		ObjectMeta: v1.ObjectMeta{Name: "Test Task"},
		Spec: pipelinev1.TaskSpec{
			Steps: []pipelinev1.Step{pipelinev1.Step{}},
			Volumes: []corev1.Volume{corev1.Volume{
				Name: defaults.SubstitutionRadixGitDeployKeyTarget,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: operatorDefaults.GitPrivateKeySecretName,
					},
				},
			}},
		},
	})

	assert.NoError(t, err)
}

func TestNoHostPathAllowed(t *testing.T) {
	err := validation.ValidateTask(&pipelinev1.Task{
		ObjectMeta: v1.ObjectMeta{Name: "Test Task"},
		Spec: pipelinev1.TaskSpec{
			Steps: []pipelinev1.Step{pipelinev1.Step{}},
			Volumes: []corev1.Volume{corev1.Volume{
				Name: "test-host-path",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/tmp",
					},
				},
			}},
		},
	})

	assert.ErrorIs(t, err, validation.ErrHostPathNotAllowed)
}

func TestCollectAllErrors(t *testing.T) {
	err := validation.ValidateTask(&pipelinev1.Task{
		ObjectMeta: v1.ObjectMeta{Name: "Test Task"},
		Spec: pipelinev1.TaskSpec{
			Volumes: []corev1.Volume{
				corev1.Volume{
					Name: "test-host-path",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/tmp",
						},
					},
				},
				corev1.Volume{
					Name: "radix-hello-world",
				},
				corev1.Volume{
					Name: "testing-secret-mount",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: "test-illegal-secret",
						},
					},
				},
			},
		},
	})

	assert.ErrorIs(t, err, validation.ErrEmptyStepList)
	assert.ErrorIs(t, err, validation.ErrHostPathNotAllowed)
	assert.ErrorIs(t, err, validation.ErrRadixVolumeNameNotAllowed)
	assert.ErrorIs(t, err, validation.ErrSecretReferenceNotAllowed)

}
