package main

import (
	"context"
	"os"

	"github.com/kubara-io/kubara/cmd"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var version = "dev" // dynamically set at build time via ldflags by GoReleaser. Defaults to "dev" for local builds.

func init() {
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05"
	log.Logger = log.Output(
		zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: zerolog.TimeFieldFormat,
		},
	)
}

func main() {
	app := cmd.NewRootCmd(version)

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Error running program")
	}
}
