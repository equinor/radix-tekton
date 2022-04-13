package pipeline

import (
	"github.com/equinor/radix-operator/pipeline-runner/steps"
	"github.com/equinor/radix-operator/pkg/apis/radix/v1"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	"github.com/equinor/radix-tekton/pkg/configmap"
	"github.com/equinor/radix-tekton/pkg/models"
	log "github.com/sirupsen/logrus"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

//Context of the pipeline
type Context interface {
	ProcessRadixAppConfig() error
}

type pipelineContext struct {
	radixClient      radixclient.Interface
	kubeClient       kubernetes.Interface
	tektonClient     tektonclient.Interface
	env              models.Env
	radixApplication *v1.RadixApplication
}

//ProcessRadixAppConfig Load radixconfig.yaml to a ConfigMap and create RadixApplication
func (ctx pipelineContext) ProcessRadixAppConfig() error {
	configFileContent, err := configmap.CreateFromFile(ctx.kubeClient, ctx.env)
	if err != nil {
		log.Fatal("Error copying radixconfig.yaml and creating config map from file: %v", err)
	}
	ctx.radixApplication, err = steps.CreateRadixApplication(ctx.radixClient, configFileContent)
	if err != nil {
		return err
	}
	log.Debugln("RA loaded")

	err = ctx.PrepareTektonPipelineJob()
	if err != nil {
		return err
	}

	return nil
}

func NewPipelineContext(kubeClient kubernetes.Interface, radixClient radixclient.Interface, env models.Env) Context {
	return &pipelineContext{
		kubeClient:  kubeClient,
		radixClient: radixClient,
		env:         env,
	}
}
