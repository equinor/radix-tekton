package main

import (
	"github.com/equinor/radix-tekton/pkg/kubernetes"
	"github.com/equinor/radix-tekton/pkg/models"
	"github.com/equinor/radix-tekton/pkg/pipeline"
	log "github.com/sirupsen/logrus"
)

func main() {
	kubeClient, radixClient, err := kubernetes.GetClients()
	if err != nil {
		log.Fatal(err.Error())
	}

	env := models.NewEnvironment()

	pipelineContext := pipeline.NewPipelineContext(kubeClient, radixClient, env)
	err = pipelineContext.ProcessRadixAppConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Infof("Successfully completed")
}
