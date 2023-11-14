package labels

import (
	"github.com/equinor/radix-operator/pkg/apis/kube"
	"github.com/equinor/radix-tekton/pkg/models"
)

// GetLabelsForEnvironment Get Pipeline object labels for a target build environment
func GetLabelsForEnvironment(ctx models.Context, targetEnv string) map[string]string {
	appName := ctx.GetConfig().GetAppName()
	imageTag := ctx.GetConfig().GetRadixImageTag()
	return map[string]string{
		kube.RadixAppLabel:      appName,
		kube.RadixEnvLabel:      targetEnv,
		kube.RadixJobNameLabel:  ctx.GetConfig().GetRadixPipelineJobName(),
		kube.RadixImageTagLabel: imageTag,
	}
}
