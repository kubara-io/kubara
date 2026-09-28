package cmd

import (
	"fmt"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/urfave/cli/v3"
)

const defaultKubeconfigPath = "~/.kube/config"

type GlobalFlags struct {
	KubeconfigFilePath string
	Catalogs           []string
	CatalogOverwrite   bool
	TestK8sConnection  bool
	DocsOutputPath     string
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
	DocsOutputPath     string
	Base64Mode         bool
	Base64             Base64Options
}

func NewGlobalFlags() *GlobalFlags {
	return &GlobalFlags{
		KubeconfigFilePath: defaultKubeconfigPath,
	}
}

func (flags *GlobalFlags) ToRootOptions() RootOptions {
	kubeconfigFilePath := flags.KubeconfigFilePath
	if kubeconfigFilePath == "" {
		kubeconfigFilePath = defaultKubeconfigPath
	}

	return RootOptions{
		KubeconfigFilePath: kubeconfigFilePath,
		TestK8sConnection:  flags.TestK8sConnection,
		CheckUpdateFlag:    flags.CheckUpdateFlag,
		DocsOutputPath:     flags.DocsOutputPath,
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
			Sources:     cli.EnvVars("KUBECONFIG"),
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
		&cli.StringSliceFlag{
			Name:        "catalog",
			Value:       flags.Catalogs,
			Usage:       "Path to a catalog directory or an OCI reference in the form oci://registry/repository:x.y.z",
			Destination: &flags.Catalogs,
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
		&cli.BoolFlag{
			Name:        "catalog-overwrite",
			Value:       flags.CatalogOverwrite,
			Usage:       "Allow later catalogs from --catalog to overwrite earlier definitions on name collisions",
			Destination: &flags.CatalogOverwrite,
		},
		&cli.BoolFlag{
			Name:        "test-connection",
			Value:       flags.TestK8sConnection,
			Usage:       "Check if Kubernetes cluster can be reached. List namespaces and exit",
			Destination: &flags.TestK8sConnection,
		},
		&cli.StringFlag{
			Name:        "docs",
			Value:       flags.DocsOutputPath,
			Usage:       "Output file path for generated command docs",
			Destination: &flags.DocsOutputPath,
			Hidden:      true,
			Config: cli.StringConfig{
				TrimSpace: true,
			},
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
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
		&cli.StringFlag{
			Name:        "file",
			Value:       flags.InputFile,
			Usage:       "Input file path for base64 operation",
			Destination: &flags.InputFile,
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
		&cli.BoolFlag{
			Name:        "check-update",
			Value:       flags.CheckUpdateFlag,
			Usage:       "Check online for a newer kubara release",
			Destination: &flags.CheckUpdateFlag,
		},
	}
}

type WorkspacePaths = utils.WorkspacePaths

func ResolveWorkspace(cmd *cli.Command) (*WorkspacePaths, error) {
	return utils.ResolveWorkspaceFromCommand(cmd)
}

func catalogLoadOptionsFromCommand(cmd *cli.Command, bootstrapCatalog string) (catalog.LoadOptions, error) {
	ws, err := ResolveWorkspace(cmd)
	if err != nil {
		return catalog.LoadOptions{}, fmt.Errorf("resolve workspace: %w", err)
	}

	return catalog.ResolveLoadOptions(ws.WorkDir, bootstrapCatalog, cmd.StringSlice("catalog"), cmd.Bool("catalog-overwrite"))
}
