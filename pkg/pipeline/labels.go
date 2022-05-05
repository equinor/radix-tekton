package pipeline

import (
	"fmt"
	"github.com/equinor/radix-operator/pkg/apis/kube"
)

func (ctx pipelineContext) getLabels(targetEnv string) map[string]string {
	appName := ctx.env.GetAppName()
	imageTag := ctx.env.GetRadixImageTag()
	return map[string]string{
		kube.RadixJobNameLabel:     ctx.env.GetRadixPipelineJobName(),
		kube.RadixBuildLabel:       fmt.Sprintf("%s-%s-%s", appName, imageTag, ctx.hash),
		kube.RadixAppLabel:         appName,
		kube.RadixEnvLabel:         targetEnv,
		kube.RadixImageTagLabel:    imageTag,
		kube.RadixJobTypeLabel:     kube.RadixJobTypeBuild,
		kube.RadixPipelineRunLabel: ctx.env.GetRadixPipelineRun(),
	}
}
