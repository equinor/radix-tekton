package pipeline

import (
    "github.com/equinor/radix-operator/pipeline-runner/steps"
    "github.com/equinor/radix-operator/pkg/apis/radix/v1"
    radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
    "github.com/equinor/radix-tekton/pkg/configmap"
    log "github.com/sirupsen/logrus"
    "k8s.io/client-go/kubernetes"
)

//Context of the pipeline
type Context interface {
    ProcessRadixConfig() (*v1.RadixApplication, error)
}

type pipelineContext struct {
    namespace     string
    configMapName string
    file          string
    pipelineName  string
    radixClient   radixclient.Interface
    kubeClient    kubernetes.Interface
}

//ProcessRadixConfig Load radixconfig.yaml to a ConfigMap and create RadixApplication
func (ctx pipelineContext) ProcessRadixConfig() (*v1.RadixApplication, error) {
    log.Infof("Copying radixconfig.yaml file (%s) into namespace (%s) and configmap (%s)", ctx.file, ctx.namespace,
        ctx.configMapName)
    configFileContent, err := configmap.CreateFromFile(ctx.kubeClient, ctx.namespace, ctx.configMapName, ctx.file)
    if err != nil {
        log.Fatal("Error copying radixconfig.yaml and creating config map from file: %v", err)
    }
    ra, err := steps.CreateRadixApplication(ctx.radixClient, configFileContent)
    if err != nil {
        return nil, err
    }
    if ra != nil {
        log.Info("RA loaded")
    }
    return ra, nil
}

func NewPipelineContext(kubeClient kubernetes.Interface, radixClient radixclient.Interface, namespace, configMapName,
    file, pipelineName string) Context {
    return &pipelineContext{
        kubeClient:    kubeClient,
        radixClient:   radixClient,
        namespace:     namespace,
        configMapName: configMapName,
        file:          file,
        pipelineName:  pipelineName,
    }
}
