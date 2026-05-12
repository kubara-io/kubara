package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kubara-io/kubara/internal/bootstrap"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/render"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/urfave/cli/v3"
)

type BootstrapFlags struct {
	WithES                 bool
	WithProm               bool
	ClusterSecretStorePath string
	ManagedCatalogPath     string
	OverlayValuesPath      string
	EnvFile                string
	EnvPrefixFlag          string
	DryRun                 bool
	Timeout                time.Duration
}

func NewBootstrapFlags() *BootstrapFlags {
	return &BootstrapFlags{
		WithES:        true,
		WithProm:      true,
		EnvFile:       ".env",
		EnvPrefixFlag: "KUBARA_",
		Timeout:       2 * time.Minute,
	}
}

func NewBootstrapCmd() *cli.Command {
	flags := NewBootstrapFlags()

	cmd := &cli.Command{
		Name:      "bootstrap",
		Usage:     "Bootstrap ArgoCD onto the specified cluster with optional external-secrets and prometheus CRD",
		ArgsUsage: "(cluster-name)",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:      "cluster-name",
				UsageText: "The name of the cluster as set in the config",
			},
		},
		Action: func(c context.Context, cmd *cli.Command) error {
			o, err := flags.ToOptions(cmd)
			if err != nil {
				return fmt.Errorf("convert flags to options: %w", err)
			}
			if cmd.StringArg("cluster-name") == "" {
				return fmt.Errorf("missing argument %q", "cluster-name")
			}
			o.ClusterName = cmd.StringArg("cluster-name")
			return Run(c, o)
		},
	}
	flags.AddFlags(cmd)

	return cmd
}

func (flags *BootstrapFlags) ToOptions(cmd *cli.Command) (*bootstrap.Options, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	envFilePath, err := utils.GetFullPath(cmd.String("env-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get env file path: %w", err)
	}

	kubeconf, err := utils.GetFullPath(cmd.String("kubeconfig"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get kubeconfig path: %w", err)
	}

	managedAbsPath := flags.ManagedCatalogPath
	if !filepath.IsAbs(managedAbsPath) {
		managedAbsPath = filepath.Join(cwd, managedAbsPath)
		managedAbsPath, err = filepath.Abs(managedAbsPath)
		if err != nil {
			return nil, fmt.Errorf("resolve absolute path: %w", err)
		}
	}

	customerAbsPath := flags.OverlayValuesPath
	if !filepath.IsAbs(customerAbsPath) {
		customerAbsPath = filepath.Join(cwd, customerAbsPath)
		customerAbsPath, err = filepath.Abs(customerAbsPath)
		if err != nil {
			return nil, fmt.Errorf("resolve absolute path: %w", err)
		}
	}

	catalogOptions, err := catalogLoadOptionsFromCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("get catalog options: %w", err)
	}

	// Load environment
	es := envconfig.NewEnvStore(envFilePath, ".", flags.EnvPrefixFlag)
	if err := es.Load(); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}
	if err := es.ValidateAll(); err != nil {
		return nil, fmt.Errorf("validate env: %w", err)
	}

	envMap := es.GetConfig()

	// Load config file and find cluster by name
	configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get config file path: %w", err)
	}

	cs := config.NewConfigStoreWithCatalog(configFilePath, catalogOptions)
	if err := cs.Load(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if err := cs.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	// Find the cluster by name from the argument
	clusterName := cmd.StringArg("cluster-name")
	var clusterConfig *config.Cluster
	for i := range cs.GetConfig().Clusters {
		if cs.GetConfig().Clusters[i].Name == clusterName {
			clusterConfig = &cs.GetConfig().Clusters[i]
			break
		}
	}
	if clusterConfig == nil {
		return nil, fmt.Errorf("cluster %q not found in config file %q", clusterName, configFilePath)
	}

	// Validate and normalize ClusterSecretStore path if provided
	var cssAbsPath string
	if flags.ClusterSecretStorePath != "" {
		if !filepath.IsAbs(flags.ClusterSecretStorePath) {
			cssAbsPath = filepath.Join(cwd, flags.ClusterSecretStorePath)
			cssAbsPath, err = filepath.Abs(cssAbsPath)
			if err != nil {
				return nil, fmt.Errorf("getting absolute path for ClusterSecretStore file: %w", err)
			}
		} else {
			cssAbsPath = flags.ClusterSecretStorePath
		}

		// Verify file exists
		if _, err := os.Stat(cssAbsPath); err != nil {
			return nil, fmt.Errorf("cluster secret store file not found: %w", err)
		}
	}

	catalog, err := cs.GetCatalog()
	if err != nil {
		return nil, fmt.Errorf("load catalog: %w", err)
	}

	return &bootstrap.Options{
		Kubeconfig:     kubeconf,
		ManagedCatalog: managedAbsPath,
		OverlayValues:  customerAbsPath,
		WithES:         flags.WithES,
		WithProm:       flags.WithProm,
		WithESCSSPath:  cssAbsPath,
		EnvMap:         envMap,
		Catalog:        catalog,
		ClusterConfig:  clusterConfig,
		DryRun:         flags.DryRun,
		Timeout:        flags.Timeout,
		ClusterName:    clusterName,
	}, nil
}

func (flags *BootstrapFlags) AddFlags(cmd *cli.Command) {
	bootstrapFlags := []cli.Flag{
		// TODO: Implement dry-run with kubernetes client
		&cli.BoolFlag{
			Name:        "dry-run",
			Value:       false,
			Usage:       "Run with dry-run",
			Destination: &flags.DryRun,
		},
		&cli.BoolFlag{
			Name:        "with-es-crds",
			Usage:       "Also install external-secrets",
			Destination: &flags.WithES,
		},
		&cli.BoolFlag{
			Name:        "with-prometheus-crds",
			Usage:       "Also install kube-prometheus-stack",
			Destination: &flags.WithProm,
		},
		&cli.StringFlag{
			Name:        "with-es-css-file",
			Usage:       "Path to the ClusterSecretStore manifest file (supports go-template + sprig)",
			Destination: &flags.ClusterSecretStorePath,
		},
		&cli.StringFlag{
			Name:        "managed-catalog",
			Value:       render.DefaultManagedCatalogPath,
			Usage:       "Path to the managed catalog directory",
			Destination: &flags.ManagedCatalogPath,
		},
		&cli.StringFlag{
			Name:        "overlay-values",
			Value:       render.DefaultOverlayValuesPath,
			Usage:       "Path to overlay values directory",
			Destination: &flags.OverlayValuesPath,
		},
		&cli.StringFlag{
			Name:        "envVarPrefix",
			Value:       flags.EnvPrefixFlag,
			Usage:       "Prefix for envs read from envVars",
			Destination: &flags.EnvPrefixFlag,
		},
		&cli.DurationFlag{
			Name:        "timeout",
			Value:       5 * time.Minute,
			Usage:       "Timeout for kubernetes API calls (e.g. 10s, 1m)",
			Destination: &flags.Timeout,
		},
	}

	cmd.Flags = append(cmd.Flags, bootstrapFlags...)
}

func Run(ctx context.Context, o *bootstrap.Options) error {
	ctx, cancelSignal := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancelSignal()

	return bootstrap.Bootstrap(ctx, o)
}
