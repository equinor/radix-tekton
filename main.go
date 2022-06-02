package main

import (
	"fmt"

	pipelineDefaults "github.com/equinor/radix-operator/pipeline-runner/model/defaults"
	"github.com/equinor/radix-tekton/pkg/kubernetes"
	"github.com/equinor/radix-tekton/pkg/models"
	"github.com/equinor/radix-tekton/pkg/models/env"
	"github.com/equinor/radix-tekton/pkg/pipeline"
	log "github.com/sirupsen/logrus"
)

func main() {
	environment := env.NewEnvironment()
	setLogLevel(environment)

	kubeClient, radixClient, tektonClient, err := kubernetes.GetClients()
	if err != nil {
		log.Fatal(err.Error())
	}

	ctx := pipeline.NewPipelineContext(kubeClient, radixClient, tektonClient, environment)
	err = runAction(ctx)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Infof("Completed")
}

func setLogLevel(environment env.Env) {
	logLevel := environment.GetLogLevel()
	log.SetLevel(logLevel)
	log.Debugf("log-level '%v'", logLevel)
}
func runAction(ctx models.Context) error {
	action := ctx.GetEnv().GetPipelinesAction()
	log.Infof("execute an action %s", action)
	switch action {
	case pipelineDefaults.RadixPipelineActionPrepare:
		return ctx.ProcessRadixAppConfig()
	case pipelineDefaults.RadixPipelineActionRun:
		return ctx.RunPipelinesJob()
	default:
		return fmt.Errorf("unsupported action %s", action)
	}
}
