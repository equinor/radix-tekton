package main

import (
	"context"
	"fmt"
	"os"
	"time"

	pipelineDefaults "github.com/equinor/radix-operator/pipeline-runner/model/defaults"
	"github.com/equinor/radix-tekton/pkg/kubernetes"
	"github.com/equinor/radix-tekton/pkg/models"
	"github.com/equinor/radix-tekton/pkg/models/env"
	"github.com/equinor/radix-tekton/pkg/pipeline"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	environment := env.NewEnvironment()
	_, err := initializeLogger(context.Background(), environment)
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

func initializeLogger(ctx context.Context, environment env.Env) (context.Context, error) {
	logLevel, err := environment.GetLogLevel()
	if err != nil {
		return ctx, err
	}

	zerolog.SetGlobalLevel(logLevel)
	zerolog.DurationFieldUnit = time.Millisecond
	if environment.GetPrettyPrint() {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.TimeOnly})
	}
	ctx = log.Logger.WithContext(ctx)

	log.Debug().Msgf("log-level '%v'", logLevel.String())
	return ctx, nil
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
