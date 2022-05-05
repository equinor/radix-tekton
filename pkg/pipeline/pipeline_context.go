package pipeline

import (
	log "github.com/sirupsen/logrus"
	"strings"

	"github.com/equinor/radix-common/utils"
	"github.com/equinor/radix-operator/pkg/apis/radix/v1"
	radixclient "github.com/equinor/radix-operator/pkg/client/clientset/versioned"
	"github.com/equinor/radix-tekton/pkg/models/env"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

//Context of the pipeline
type Context interface {
	ProcessRadixAppConfig() error
	RunTektonPipelineJob() error
	GetEnv() env.Env
}

type pipelineContext struct {
	radixClient        radixclient.Interface
	kubeClient         kubernetes.Interface
	tektonClient       tektonclient.Interface
	env                env.Env
	radixApplication   *v1.RadixApplication
	targetEnvironments map[string]bool
	hash               string
}

func (ctx *pipelineContext) GetEnv() env.Env {
	return ctx.env
}

func (ctx *pipelineContext) GetKubeClient() kubernetes.Interface {
	return ctx.kubeClient
}

func (ctx *pipelineContext) getEnvVars(string) v1.EnvVarsMap {
	envVarsMap := make(v1.EnvVarsMap)
	if ctx.radixApplication.Spec.Build != nil && ctx.radixApplication.Spec.Build.Variables != nil && len(ctx.radixApplication.Spec.Build.Variables) > 0 {
		for name, envVar := range ctx.radixApplication.Spec.Build.Variables {
			envVarsMap[name] = envVar
		}
		log.Debugf("Loaded %d radixApplication build variables", len(ctx.radixApplication.Spec.Build.Variables))
	}
	return envVarsMap
}

func NewPipelineContext(kubeClient kubernetes.Interface, radixClient radixclient.Interface, tektonClient tektonclient.Interface, environment env.Env) Context {
	return &pipelineContext{
		kubeClient:   kubeClient,
		radixClient:  radixClient,
		tektonClient: tektonClient,
		env:          environment,
		hash:         strings.ToLower(utils.RandStringStrSeed(5, environment.GetRadixPipelineRun())),
	}
}
