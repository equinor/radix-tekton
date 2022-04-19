package main

import (
	"fmt"

	"github.com/equinor/radix-tekton/pkg/kubernetes"
	"github.com/equinor/radix-tekton/pkg/models/env"
	"github.com/equinor/radix-tekton/pkg/pipeline"
	log "github.com/sirupsen/logrus"
)

func main() {
	kubeClient, radixClient, tektonClient, err := kubernetes.GetClients()
	if err != nil {
		log.Fatal(err.Error())
	}

	env := env.NewEnvironment()

	ctx := pipeline.NewPipelineContext(kubeClient, radixClient, tektonClient, env)

	if err != nil {
		log.Fatal(err.Error())
	}
	err = runAction(ctx)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Infof("Successfully completed")
}
func runAction(ctx pipeline.Context) error {
	action := ctx.GetEnv().GetTektonAction()
	switch action {
	case "prepare":
		return ctx.ProcessRadixAppConfig()
	case "run":
		return ctx.RunTektonPipelineJob()
	default:
		return fmt.Errorf("unsupported action '%s'", action)
	}
}
