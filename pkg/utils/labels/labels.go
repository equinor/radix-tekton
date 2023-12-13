package labels

import (
	"github.com/equinor/radix-operator/pkg/apis/kube"
	"github.com/equinor/radix-tekton/pkg/models"
)

const (
	AzureWorkloadIdentityUse = "azure.workload.identity/use"
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
