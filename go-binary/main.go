package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"kubara/cmd"
	"kubara/internal/updatecheck"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

const (
	AppName = "kubara"
)

var authors = []any{
	"Contributors: https://github.com/kubara-io/kubara/graphs/contributors"}

var (
	version = "dev" //version is dynamically set at build time via ldflags by GoReleaser. Defaults to "dev" for local builds.
)

func init() {
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

var (
	kubeconfigFilePath string
	testK8sConnection  bool
	checkUpdateFlag    bool
	catalogPath        string
	catalogOverwrite   bool
	base64Mode         bool
	encodeFlag         bool
	decodeFlag         bool
	inputFile          string
	inputString        string
)

func NewAppAction(cmd *cli.Command) error {
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

func main() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "kubeconfig",
			Value:       "~/.kube/config",
			Usage:       "Path to kubeconfig file",
			Destination: &kubeconfigFilePath,
		},
		&cli.StringFlag{
			Name:    "work-dir",
			Aliases: []string{"w"},
			Value:   ".",
			Usage:   "Working directory",
		},
		&cli.StringFlag{
			Name:    "config-file",
			Aliases: []string{"c"},
			Value:   "config.yaml",
			Usage:   "Path to the configuration file",
		},

		&cli.StringFlag{
			Name:  "env-file",
			Value: ".env",
			Usage: "Path to the .env file",
		},
		&cli.StringFlag{
			Name:        "catalog",
			Value:       "",
			Usage:       "Path to external ServiceDefinition catalog directory.",
			Destination: &catalogPath,
		},
		&cli.BoolFlag{
			Name:        "catalog-overwrite",
			Value:       false,
			Usage:       "Allow external service definitions from --catalog to overwrite built-in definitions on name collisions.",
			Destination: &catalogOverwrite,
		},
		&cli.BoolFlag{
			Name:        "test-connection",
			Value:       false,
			Usage:       "Check if Kubernetes cluster can be reached. List namespaces and exit",
			Destination: &testK8sConnection,
		},
		&cli.BoolFlag{
			Name:        "base64",
			Value:       false,
			Usage:       "Enable base64 encode/decode mode",
			Destination: &base64Mode,
		},
		&cli.BoolFlag{
			Name:        "encode",
			Value:       false,
			Usage:       "Base64 encode input",
			Destination: &encodeFlag,
		}, &cli.BoolFlag{
			Name:        "decode",
			Value:       false,
			Usage:       "Base64 decode input",
			Destination: &decodeFlag,
		},
		&cli.StringFlag{
			Name:        "string",
			Value:       "",
			Usage:       "Input string for base64 operation",
			Destination: &inputString,
		},
		&cli.StringFlag{
			Name:        "file",
			Value:       "",
			Usage:       "Input file path for base64 operation",
			Destination: &inputFile,
		},
		&cli.BoolFlag{
			Name:        "check-update",
			Value:       false,
			Usage:       "Check online for a newer kubara release",
			Destination: &checkUpdateFlag,
		},
	}

	app := &cli.Command{
		Name:        AppName,
		Version:     version,
		Authors:     authors,
		Copyright:   "",
		Usage:       "Opinionated CLI for Kubernetes platform engineering",
		Flags:       flags,
		UsageText:   "",
		Description: "kubara is an opinionated CLI to bootstrap and operate Kubernetes platforms with GitOps-first workflows.",
		Commands: []*cli.Command{
			cmd.NewInitCmd(),
			cmd.NewGenerateCmd(),
			cmd.NewBootstrapCmd(),
			cmd.NewSchemaCmd(),
		},
		Action: func(cCtx context.Context, cmd *cli.Command) error {
			return NewAppAction(cmd)
		},
	}

	if !slices.Contains(os.Args[1:], "--check-update") {
		updatecheck.NotifyIfNewReleaseAvailable(version, os.Stderr)
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Error running program")
	}

}
