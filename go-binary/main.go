package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"kubara/cmd"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

const (
	AppName = "kubara"
)

var authors = []any{
	mail.Address{
		Name:    "Alexander Hoeft",
		Address: "alexander.hoeft@iits-consulting.de",
	},
	mail.Address{
		Name:    "Artem Lajko",
		Address: "artem.lajko@iits-consulting.de",
	},
	mail.Address{
		Name:    "Matthias Huether",
		Address: "matthias.huether@iits-consulting.de",
	},
	mail.Address{
		Name:    "Fabian Schmitt",
		Address: "fabian-patrice.schmitt@stackit.cloud",
	},
}

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
	}

	app := &cli.Command{
		Name:        AppName,
		Version:     version,
		Authors:     authors,
		Copyright:   "",
		Usage:       "",
		Flags:       flags,
		UsageText:   "",
		Description: "",
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
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal().Err(err).Msg("Error running program")
	}

}
