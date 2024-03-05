package main

import (
	"context"
	"fmt"

	pipelineDefaults "github.com/equinor/radix-operator/pipeline-runner/model/defaults"
	"github.com/equinor/radix-tekton/pkg/kubernetes"
	"github.com/equinor/radix-tekton/pkg/models"
	"github.com/equinor/radix-tekton/pkg/models/env"
	"github.com/equinor/radix-tekton/pkg/pipeline"
	"github.com/equinor/radix-tekton/pkg/utils/logger"
	"github.com/rs/zerolog/log"
)

func main() {
	environment := env.NewEnvironment()
	level, err := environment.GetLogLevel()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize log level")
	}
	logger.InitializeLogger(context.Background(), level, true)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	kubeClient, radixClient, tektonClient, err := kubernetes.GetClients()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize kubeClient")
	}

	ctx := pipeline.NewPipelineContext(kubeClient, radixClient, tektonClient, environment)
	err = runAction(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to run ")
	}

	log.Info().Msg("Completed")
}

func runAction(ctx models.Context) error {
	action := ctx.GetEnv().GetPipelinesAction()
	log.Debug().Msgf("execute an action %s", action)
	switch action {
	case pipelineDefaults.RadixPipelineActionPrepare:
		return ctx.ProcessRadixAppConfig()
	case pipelineDefaults.RadixPipelineActionRun:
		return ctx.RunPipelinesJob()
	default:
		return fmt.Errorf("unsupported action %s", action)
	}
}
