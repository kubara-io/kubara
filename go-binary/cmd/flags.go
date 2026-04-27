package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/urfave/cli/v3"
)

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

// globalFlags returns the top-level flags shared by the root command and tests.
func globalFlags() []cli.Flag {
	return []cli.Flag{
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
		},
		&cli.BoolFlag{
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
}

func catalogLoadOptionsFromCommand(cmd *cli.Command) (catalog.LoadOptions, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return catalog.LoadOptions{}, fmt.Errorf("failed to get working directory: %w", err)
	}

	rawCatalogPath := strings.TrimSpace(cmd.String("catalog"))
	if rawCatalogPath == "" {
		return catalog.LoadOptions{
			CatalogPath: "",
			Overwrite:   cmd.Bool("catalog-overwrite"),
		}, nil
	}

	absoluteCatalogPath, err := utils.GetFullPath(rawCatalogPath, cwd)
	if err != nil {
		return catalog.LoadOptions{}, fmt.Errorf("failed to get catalog path: %w", err)
	}

	return catalog.LoadOptions{
		CatalogPath: absoluteCatalogPath,
		Overwrite:   cmd.Bool("catalog-overwrite"),
	}, nil
}
