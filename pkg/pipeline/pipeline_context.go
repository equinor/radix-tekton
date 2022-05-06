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

func (ctx *pipelineContext) getEnvVars(targetEnv string) v1.EnvVarsMap {
	envVarsMap := make(v1.EnvVarsMap)
	ctx.setPipelineRunParamsFromBuild(envVarsMap)
	ctx.setPipelineRunParamsFromEnvironmentBuilds(targetEnv, envVarsMap)
	return envVarsMap
}

func (ctx *pipelineContext) setPipelineRunParamsFromBuild(envVarsMap v1.EnvVarsMap) {
	if ctx.radixApplication.Spec.Build == nil ||
		ctx.radixApplication.Spec.Build.Variables == nil ||
		len(ctx.radixApplication.Spec.Build.Variables) == 0 {
		log.Debugln("No radixApplication build variables")
		return
	}

	for name, envVar := range ctx.radixApplication.Spec.Build.Variables {
		envVarsMap[name] = envVar
	}
}

func (ctx *pipelineContext) setPipelineRunParamsFromEnvironmentBuilds(targetEnv string, envVarsMap v1.EnvVarsMap) {
	for _, env := range ctx.radixApplication.Spec.Environments {
		if !strings.EqualFold(env.Name, targetEnv) || env.Build.Variables == nil {
			continue
		}
		for envVarName, envVarVal := range env.Build.Variables {
			envVarsMap[envVarName] = envVarVal //Overrides common env-vars from Spec.Build, if any
		}
	}
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
