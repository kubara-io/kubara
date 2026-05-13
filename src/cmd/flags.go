package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/urfave/cli/v3"
)

const defaultKubeconfigPath = "~/.kube/config"

type GlobalFlags struct {
	KubeconfigFilePath string
	WorkDir            string
	ConfigFilePath     string
	EnvFilePath        string
	CatalogPath        string
	CatalogOverwrite   bool
	TestK8sConnection  bool
	Base64Mode         bool
	EncodeFlag         bool
	DecodeFlag         bool
	InputFile          string
	InputString        string
	CheckUpdateFlag    bool
}

type Base64Options struct {
	Encode      bool
	Decode      bool
	InputFile   string
	InputString string
}

type RootOptions struct {
	KubeconfigFilePath string
	TestK8sConnection  bool
	CheckUpdateFlag    bool
	Base64Mode         bool
	Base64             Base64Options
}

func NewGlobalFlags() *GlobalFlags {
	return &GlobalFlags{
		KubeconfigFilePath: defaultKubeconfigPath,
		WorkDir:            ".",
		ConfigFilePath:     "config.yaml",
		EnvFilePath:        ".env",
	}
}

func (flags *GlobalFlags) ToRootOptions() RootOptions {
	kubeconfigFilePath := flags.KubeconfigFilePath
	if kubeconfigFilePath == defaultKubeconfigPath {
		if envKC := os.Getenv("KUBECONFIG"); envKC != "" {
			kubeconfigFilePath = envKC
		}
	}

	return RootOptions{
		KubeconfigFilePath: kubeconfigFilePath,
		TestK8sConnection:  flags.TestK8sConnection,
		CheckUpdateFlag:    flags.CheckUpdateFlag,
		Base64Mode:         flags.Base64Mode,
		Base64: Base64Options{
			Encode:      flags.EncodeFlag,
			Decode:      flags.DecodeFlag,
			InputFile:   flags.InputFile,
			InputString: flags.InputString,
		},
	}
}

// globalFlags returns the top-level flags shared by the root command and tests.
func (flags *GlobalFlags) CLIFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "kubeconfig",
			Value:       flags.KubeconfigFilePath,
			Usage:       "Path to kubeconfig file",
			Destination: &flags.KubeconfigFilePath,
		},
		&cli.StringFlag{
			Name:        "work-dir",
			Aliases:     []string{"w"},
			Value:       flags.WorkDir,
			Usage:       "Working directory",
			Destination: &flags.WorkDir,
		},
		&cli.StringFlag{
			Name:        "config-file",
			Aliases:     []string{"c"},
			Value:       flags.ConfigFilePath,
			Usage:       "Path to the configuration file",
			Destination: &flags.ConfigFilePath,
		},
		&cli.StringFlag{
			Name:        "env-file",
			Value:       flags.EnvFilePath,
			Usage:       "Path to the .env file",
			Destination: &flags.EnvFilePath,
		},
		&cli.StringFlag{
			Name:        "catalog",
			Value:       flags.CatalogPath,
			Usage:       "Path to external ServiceDefinition catalog directory.",
			Destination: &flags.CatalogPath,
		},
		&cli.BoolFlag{
			Name:        "catalog-overwrite",
			Value:       flags.CatalogOverwrite,
			Usage:       "Allow external service definitions from --catalog to overwrite built-in definitions on name collisions.",
			Destination: &flags.CatalogOverwrite,
		},
		&cli.BoolFlag{
			Name:        "test-connection",
			Value:       flags.TestK8sConnection,
			Usage:       "Check if Kubernetes cluster can be reached. List namespaces and exit",
			Destination: &flags.TestK8sConnection,
		},
		&cli.BoolFlag{
			Name:        "base64",
			Value:       flags.Base64Mode,
			Usage:       "Enable base64 encode/decode mode",
			Destination: &flags.Base64Mode,
		},
		&cli.BoolFlag{
			Name:        "encode",
			Value:       flags.EncodeFlag,
			Usage:       "Base64 encode input",
			Destination: &flags.EncodeFlag,
		},
		&cli.BoolFlag{
			Name:        "decode",
			Value:       flags.DecodeFlag,
			Usage:       "Base64 decode input",
			Destination: &flags.DecodeFlag,
		},
		&cli.StringFlag{
			Name:        "string",
			Value:       flags.InputString,
			Usage:       "Input string for base64 operation",
			Destination: &flags.InputString,
		},
		&cli.StringFlag{
			Name:        "file",
			Value:       flags.InputFile,
			Usage:       "Input file path for base64 operation",
			Destination: &flags.InputFile,
		},
		&cli.BoolFlag{
			Name:        "check-update",
			Value:       flags.CheckUpdateFlag,
			Usage:       "Check online for a newer kubara release",
			Destination: &flags.CheckUpdateFlag,
		},
	}
}

func catalogLoadOptionsFromCommand(cmd *cli.Command) (catalog.LoadOptions, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return catalog.LoadOptions{}, fmt.Errorf("get working directory: %w", err)
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
		return catalog.LoadOptions{}, fmt.Errorf("get catalog path: %w", err)
	}

	return catalog.LoadOptions{
		CatalogPath: absoluteCatalogPath,
		Overwrite:   cmd.Bool("catalog-overwrite"),
	}, nil
}
