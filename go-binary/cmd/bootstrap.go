package cmd

import (
	"context"
	"fmt"
	"kubara/assets/catalog"
	"kubara/templates"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"kubara/assets/config"
	"kubara/assets/envmap"
	"kubara/internal/bootstrap"
	"kubara/utils"

	"github.com/urfave/cli/v3"
)

type BootstrapFlags struct {
	WithES                 bool
	WithProm               bool
	ClusterSecretStorePath string
	ManagedCatalogPath     string
	OverlayValuesPath      string
	CatalogPath            string
	CatalogOverwrite       bool
	EnvFile                string
	EnvPrefixFlag          string
	DryRun                 bool
	Timeout                time.Duration
}

func NewBootstrapFlags() *BootstrapFlags {
	return &BootstrapFlags{
		WithES:           true,
		WithProm:         true,
		CatalogPath:      "",
		CatalogOverwrite: false,
		EnvFile:          ".env",
		EnvPrefixFlag:    "KUBARA_",
		Timeout:          2 * time.Minute,
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
				return fmt.Errorf("couldn't convert flags to options: %w", err)
			}
			if cmd.StringArg("cluster-name") == "" {
				return fmt.Errorf("missing argument %s", "cluster-name")
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
		return nil, err
	}

	envFilePath, err := utils.GetFullPath(cmd.String("env-file"), cwd)
	if err != nil {
		return nil, err
	}

	kubeconf, err := utils.GetFullPath(cmd.String("kubeconfig"), cwd)
	if err != nil {
		return nil, err
	}

	managedAbsPath := flags.ManagedCatalogPath
	if !filepath.IsAbs(managedAbsPath) {
		managedAbsPath = filepath.Join(cwd, managedAbsPath)
		managedAbsPath, err = filepath.Abs(managedAbsPath)
		if err != nil {
			return nil, fmt.Errorf("getting absoulte Path failed: %w", err)
		}
	}

	customerAbsPath := flags.OverlayValuesPath
	if !filepath.IsAbs(customerAbsPath) {
		customerAbsPath = filepath.Join(cwd, customerAbsPath)
		customerAbsPath, err = filepath.Abs(customerAbsPath)
		if err != nil {
			return nil, fmt.Errorf("getting absoulte Path failed: %w", err)
		}
	}

	catalogPath := ""
	if flags.CatalogPath != "" {
		catalogPath, err = utils.GetFullPath(flags.CatalogPath, cwd)
		if err != nil {
			return nil, fmt.Errorf("getting catalog path failed: %w", err)
		}
	}

	// Load environment
	em := envmap.NewEnvMapManager(envFilePath, ".", flags.EnvPrefixFlag)
	if err := em.Load(); err != nil {
		return nil, fmt.Errorf("reading Env failed: %w", err)
	}
	if err := em.ValidateAll(); err != nil {
		return nil, fmt.Errorf("validating env: %w", err)
	}

	envMap := em.GetConfig()

	// Load config file and find cluster by name
	configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("getting config file path: %w", err)
	}

	cm := config.NewConfigManagerWithCatalog(configFilePath, catalog.LoadOptions{
		DistributionPath: catalogPath,
		Overwrite:        flags.CatalogOverwrite,
	})
	if err := cm.Load(); err != nil {
		return nil, fmt.Errorf("loading config from %s: %w", configFilePath, err)
	}
	if err := cm.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	// Find the cluster by name from the argument
	clusterName := cmd.StringArg("cluster-name")
	var clusterConfig *config.Cluster
	for i := range cm.GetConfig().Clusters {
		if cm.GetConfig().Clusters[i].Name == clusterName {
			clusterConfig = &cm.GetConfig().Clusters[i]
			break
		}
	}
	if clusterConfig == nil {
		return nil, fmt.Errorf("cluster '%s' not found in config file %s", clusterName, configFilePath)
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
			return nil, fmt.Errorf("ClusterSecretStore file not found: %w", err)
		}
	}

	return &bootstrap.Options{
		Kubeconfig:     kubeconf,
		ManagedCatalog: managedAbsPath,
		OverlayValues:  customerAbsPath,
		WithES:         flags.WithES,
		WithProm:       flags.WithProm,
		WithESCSSPath:  cssAbsPath,
		EnvMap:         envMap,
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
			Value:       templates.DefaultManagedCatalogPath,
			Usage:       "Path to the managed catalog directory",
			Destination: &flags.ManagedCatalogPath,
		},
		&cli.StringFlag{
			Name:        "overlay-values",
			Value:       templates.DefaultOverlayValuesPath,
			Usage:       "Path to overlay values directory",
			Destination: &flags.OverlayValuesPath,
		},
		&cli.StringFlag{
			Name:        "catalog",
			Value:       flags.CatalogPath,
			Usage:       "Path to external ServiceDefinition catalog/distribution directory.",
			Destination: &flags.CatalogPath,
		},
		&cli.BoolFlag{
			Name:        "force",
			Aliases:     []string{"overwrite"},
			Value:       flags.CatalogOverwrite,
			Usage:       "Allow external service definitions from --catalog to overwrite built-in definitions on name collisions.",
			Destination: &flags.CatalogOverwrite,
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
