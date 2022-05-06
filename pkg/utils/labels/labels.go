package labels

import (
	"fmt"

	"github.com/equinor/radix-operator/pkg/apis/kube"
	"github.com/equinor/radix-tekton/pkg/models"
)

//GetLabelsForEnvironment Get Pipeline object labels for a target build environment
func GetLabelsForEnvironment(ctx models.Context, targetEnv string) map[string]string {
	appName := ctx.GetEnv().GetAppName()
	imageTag := ctx.GetEnv().GetRadixImageTag()
	return map[string]string{
		kube.RadixAppLabel:         appName,
		kube.RadixEnvLabel:         targetEnv,
		kube.RadixJobTypeLabel:     kube.RadixJobTypeBuild,
		kube.RadixJobNameLabel:     ctx.GetEnv().GetRadixPipelineJobName(),
		kube.RadixPipelineRunLabel: ctx.GetEnv().GetRadixPipelineRun(),
		kube.RadixBuildLabel:       fmt.Sprintf("%s-%s-%s", appName, imageTag, ctx.GetHash()),
		kube.RadixImageTagLabel:    imageTag,
	}
}
