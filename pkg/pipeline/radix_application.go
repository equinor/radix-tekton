package pipeline

import (
	"context"

	"github.com/equinor/radix-operator/pkg/apis/pipeline/application"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-tekton/pkg/utils/configmap"
)

func (ctx *pipelineContext) createRadixApplicationFromContent(configFileContent string) (*v1.RadixApplication, error) {
	return application.CreateRadixApplication(context.TODO(), ctx.radixClient, ctx.env.GetAppName(), ctx.env.GetDNSConfig(), configFileContent)
}

func (ctx *pipelineContext) createRadixApplicationFromConfigMap() (*v1.RadixApplication, error) {
	configFileContent, err := configmap.GetRadixConfigFromConfigMap(ctx.GetKubeClient(), ctx.env.GetAppNamespace(), ctx.env.GetRadixConfigMapName())
	if err != nil {
		return nil, err
	}
	return ctx.createRadixApplicationFromContent(configFileContent)
}
