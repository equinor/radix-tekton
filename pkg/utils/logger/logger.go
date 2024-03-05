package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func InitializeLogger(ctx context.Context, logLevel zerolog.Level, prettyPrint bool) context.Context {
	zerolog.SetGlobalLevel(logLevel)
	zerolog.DurationFieldUnit = time.Millisecond
	if prettyPrint {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.TimeOnly})
	}
	ctx = log.Logger.WithContext(ctx)

	log.Debug().Msgf("log-level '%v'", logLevel.String())
	return ctx
}
