package main

import (
	"fmt"
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
	action := ctx.GetEnv().GetTektonAction()
	log.Infof("execute an action '%s'", action)
	switch action {
	case "prepare":
		return ctx.ProcessRadixAppConfig()
	case "run":
		return ctx.RunTektonPipelineJob()
	default:
		return fmt.Errorf("unsupported action '%s'", action)
	}
}
