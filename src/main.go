package main

import (
	"context"
	"os"
	"slices"

	"github.com/kubara-io/kubara/cmd"
	"github.com/kubara-io/kubara/internal/updatecheck"
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
	if !slices.Contains(os.Args[1:], "--check-update") {
		updatecheck.NotifyIfNewReleaseAvailable(version, os.Stderr)
	}

	if err := cmd.NewRootCmd(version).Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Error running program")
	}
}
