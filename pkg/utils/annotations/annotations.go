package annotations

import (
	"errors"
	"fmt"
	"slices"

	"github.com/equinor/radix-tekton/pkg/defaults"
	"github.com/equinor/radix-tekton/pkg/pipeline/validation"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	AzureWorkloadIdentiySkipContainers = "azure.workload.identity/skip-containers"
)

var (
	AllowedAnnotations = []string{AzureWorkloadIdentiySkipContainers, defaults.PipelineTaskNameAnnotation}
)

func ValidateTaskAnnotations(task pipelinev1.Task) error {
	var errs []error

	for key, value := range task.ObjectMeta.Annotations {
		if !slices.Contains(AllowedAnnotations, key) {
			errs = append(errs, fmt.Errorf("annotation %s=%s is not allowed in task %s: %w", key, value, task.Name, validation.ErrIllegalTaskAnnotation))
		}
	}

	return errors.Join(errs...)
}
