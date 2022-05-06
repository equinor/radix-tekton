package pipeline

import (
	"github.com/equinor/radix-operator/pipeline-runner/steps"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-tekton/pkg/utils/configmap"
)

func (ctx pipelineContext) createRadixApplicationFromContent(configFileContent string) (*v1.RadixApplication, error) {
	return steps.CreateRadixApplication(ctx.radixClient, configFileContent)
}

func (ctx pipelineContext) createRadixApplicationFromConfigMap() (*v1.RadixApplication, error) {
	configFileContent, err := configmap.GetRadixConfigFromConfigMap(ctx.GetKubeClient(), ctx.env.GetAppNamespace(), ctx.env.GetConfigMapName())
	if err != nil {
		return nil, err
	}
	return ctx.createRadixApplicationFromContent(configFileContent)
}
