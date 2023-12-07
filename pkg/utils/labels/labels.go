package labels

import (
	"errors"
	"fmt"
	"slices"

	"github.com/equinor/radix-operator/pkg/apis/kube"
	"github.com/equinor/radix-tekton/pkg/models"
	"github.com/equinor/radix-tekton/pkg/pipeline/validation"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

const (
	AzureWorkloadIdenityUse = "azure.workload.identity/use"
)

var (
	AllowedLabels = []string{
		kube.RadixAppLabel,
		kube.RadixEnvLabel,
		kube.RadixJobNameLabel,
		kube.RadixImageTagLabel,
		AzureWorkloadIdenityUse,
	}
)

// GetLabelsForEnvironment Get Pipeline object labels for a target build environment
func GetLabelsForEnvironment(ctx models.Context, targetEnv string) map[string]string {
	appName := ctx.GetEnv().GetAppName()
	imageTag := ctx.GetEnv().GetRadixImageTag()
	return map[string]string{
		kube.RadixAppLabel:      appName,
		kube.RadixEnvLabel:      targetEnv,
		kube.RadixJobNameLabel:  ctx.GetEnv().GetRadixPipelineJobName(),
		kube.RadixImageTagLabel: imageTag,
	}
}

func ValidateTaskLabels(task pipelinev1.Task) error {
	var errs []error

	for key, value := range task.ObjectMeta.Labels {
		if !slices.Contains(AllowedLabels, key) {
			errs = append(errs, fmt.Errorf("label %s=%s is not allowed in task %s: %w", key, value, task.Name, validation.ErrIllegalTaskLabel))
		}
	}

	return errors.Join(errs...)
}
