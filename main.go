package main

import (
	"fmt"

	pipelineDefaults "github.com/equinor/radix-operator/pipeline-runner/model/defaults"
	"github.com/equinor/radix-tekton/pkg/kubernetes"
	"github.com/equinor/radix-tekton/pkg/models"
	"github.com/equinor/radix-tekton/pkg/models/config"
	"github.com/equinor/radix-tekton/pkg/pipeline"
	log "github.com/sirupsen/logrus"
)

func main() {
	cfg := config.NewConfig()
	setLogLevel(cfg)

	kubeClient, radixClient, tektonClient, err := kubernetes.GetClients()
	if err != nil {
		log.Fatal(err.Error())
	}

	ctx := pipeline.NewPipelineContext(kubeClient, radixClient, tektonClient, cfg)
	err = runAction(ctx)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Infof("Completed")
}

func setLogLevel(cfg config.Config) {
	logLevel := cfg.GetLogLevel()
	log.SetLevel(logLevel)
	log.Debugf("log-level '%v'", logLevel)
}
func runAction(ctx models.Context) error {
	action := ctx.GetConfig().GetPipelinesAction()
	log.Debugf("execute an action %s", action)
	switch action {
	case pipelineDefaults.RadixPipelineActionPrepare:
		return ctx.ProcessRadixAppConfig()
	case pipelineDefaults.RadixPipelineActionRun:
		return ctx.RunPipelinesJob()
	default:
		return fmt.Errorf("unsupported action %s", action)
	}
}
