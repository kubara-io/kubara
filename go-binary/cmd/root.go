package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kubara-io/kubara/internal/updatecheck"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

const AppName = "kubara"

var Authors = []any{
	"Contributors: https://github.com/kubara-io/kubara/graphs/contributors"}

var version string

func InitLogger() {
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05"
	log.Logger = log.Output(
		zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: zerolog.TimeFieldFormat,
		},
	)
}

func testConnection(kubeconfig string) {
	kc := kubeconfig
	if kc == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Err(err).Msg("home dir")
		}
		kc = filepath.Join(home, ".kube", "config")
	}
	log.Info().Msg("listing namespaces via kubectl…")
	execOrFatal(
		"kubectl",
		"--kubeconfig", kc,
		"get", "namespaces",
	)
}

func execOrFatal(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Debug().Str("cmd", fmt.Sprintf("%s %s", name, strings.Join(args, " "))).Msg("executing")
	if err := cmd.Run(); err != nil {
		log.Fatal().Err(err).Msgf("%s failed", name)
	}
}

func newAppAction(cmd *cli.Command) error {
	if kubeconfigFilePath == "~/.kube/config" {
		if envKC := os.Getenv("KUBECONFIG"); envKC != "" {
			kubeconfigFilePath = envKC
		}
	}
	// If base64 utility mode is enabled, handle it here and exit
	if base64Mode {
		if (encodeFlag && decodeFlag) || (!encodeFlag && !decodeFlag) {
			return cli.Exit("Error: specify either --encode or --decode", 1)
		}
		if (inputString != "" && inputFile != "") || (inputString == "" && inputFile == "") {
			return cli.Exit("Error: specify exactly one of --string or --file", 1)
		}
		var data []byte
		var err error
		if inputFile != "" {
			data, err = os.ReadFile(inputFile)
			if err != nil {
				log.Fatal().Err(err).Msgf("Cannot read file: %s", inputFile)
				return cli.Exit("Error: reading file", 1)
			}
		} else {
			data = []byte(inputString)
		}
		if encodeFlag {
			fmt.Print(base64.StdEncoding.EncodeToString(data))
		} else {
			decoded, err := base64.StdEncoding.DecodeString(string(data))
			if err != nil {
				log.Fatal().Err(err).Msg("Invalid base64 input")
				return cli.Exit("Error: invalid base64 input", 1)
			}
			_, err = os.Stdout.Write(decoded)
			if err != nil {
				return cli.Exit("Error: writing decoded base64 input", 1)
			}
		}
		return nil
	}

	if cmd.NumFlags() == 0 {
		cli.ShowAppHelpAndExit(cmd, 0)
	}

	switch {
	case testK8sConnection:
		testConnection(kubeconfigFilePath)
	case checkUpdateFlag:
		if err := updatecheck.PrintLiveCheck(version, os.Stdout); err != nil {
			return cli.Exit(fmt.Sprintf("Error: update check failed: %v", err), 1)
		}
	default:
		cli.ShowAppHelpAndExit(cmd, 0)
	}
	return nil
}

// NewRootCmd builds and returns the root CLI command. ver is injected from
// main via ldflags.
func NewRootCmd(ver string) *cli.Command {
	version = ver

	return &cli.Command{
		Name:        AppName,
		Version:     ver,
		Authors:     Authors,
		Copyright:   "",
		Usage:       "Opinionated CLI for Kubernetes platform engineering",
		Description: "kubara is an opinionated CLI to bootstrap and operate Kubernetes platforms with GitOps-first workflows.",
		Flags:       globalFlags(),
		Commands: []*cli.Command{
			NewInitCmd(),
			NewGenerateCmd(),
			NewBootstrapCmd(),
			NewSchemaCmd(),
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return newAppAction(cmd)
		},
	}
}
