package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/cmd/bootstrap"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/render"
	"github.com/kubara-io/kubara/internal/utils"
	"github.com/rs/zerolog/log"

	"github.com/urfave/cli/v3"
)

type BootstrapFlags struct {
	Local                  bool
	ClusterSecretStorePath string
	PlatformComponentsPath string
	PlatformConfigsPath    string
	DryRun                 bool
	Timeout                time.Duration
}

var deprecatedBootstrapCRDFlags = []string{
	"with-es-crds",
	"with-prometheus-crds",
}

func NewBootstrapFlags() *BootstrapFlags {
	return &BootstrapFlags{
		Timeout: 2 * time.Minute,
	}
}

func NewBootstrapCmd() *cli.Command {
	flags := NewBootstrapFlags()

	cmd := &cli.Command{
		Name:        "bootstrap",
		Usage:       "Bootstrap Argo CD onto a cluster",
		UsageText:   "kubara bootstrap CLUSTER_NAME [--local]",
		ArgsUsage:   "CLUSTER_NAME",
		Description: "Bootstraps Argo CD onto the specified cluster. The optional --local mode provisions an isolated local evaluation environment and is not intended for production use.",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:      "cluster-name",
				UsageText: "The cluster name defined in the config",
			},
		},
		Action: func(c context.Context, cmd *cli.Command) error {
			printDeprecatedBootstrapCRDFlagNotice(cmd)

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

	kubeconfig := cmd.String("kubeconfig")
	if strings.TrimSpace(kubeconfig) == "" {
		kubeconfig = defaultKubeconfigPath
	}
	kubeconf, err := utils.GetFullPath(kubeconfig, cwd)
	if err != nil {
		return nil, fmt.Errorf("get kubeconfig path: %w", err)
	}

	componentsAbsPath := flags.PlatformComponentsPath
	if !filepath.IsAbs(componentsAbsPath) {
		componentsAbsPath = filepath.Join(cwd, componentsAbsPath)
		componentsAbsPath, err = filepath.Abs(componentsAbsPath)
		if err != nil {
			return nil, fmt.Errorf("resolve absolute path: %w", err)
		}
	}

	configsAbsPath := flags.PlatformConfigsPath
	if !filepath.IsAbs(configsAbsPath) {
		configsAbsPath = filepath.Join(cwd, configsAbsPath)
		configsAbsPath, err = filepath.Abs(configsAbsPath)
		if err != nil {
			return nil, fmt.Errorf("resolve absolute path: %w", err)
		}
	}

	catalogOptions, err := catalogLoadOptionsFromCommand(cmd, "")
	if err != nil {
		return nil, fmt.Errorf("get catalog options: %w", err)
	}

	// Load config file and find cluster by name
	configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get config file path: %w", err)
	}

	cs := config.NewConfigStore(cwd, configFilePath, catalogOptions)
	if err := cs.Load(); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
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

	creds, err := bootstrap.LoadCredentials(cwd)
	if err != nil {
		return nil, fmt.Errorf("load credentials: %w", err)
	}

	if err := creds.ValidateForCluster(clusterConfig); err != nil {
		return nil, fmt.Errorf("validate credentials for cluster %q: %w", clusterConfig.Name, err)
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

	loadedCatalog, err := cs.GetCatalogForCluster(*clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("load catalog for cluster %q: %w", clusterConfig.Name, err)
	}

	bootstrapCatalog := catalog.DefaultBootstrapCatalog
	if cs.GetConfig() != nil && cs.GetConfig().BootstrapCatalog != nil {
		bootstrapCatalog = *cs.GetConfig().BootstrapCatalog
	}

	timeout := flags.Timeout
	if flags.Local && !cmd.IsSet("timeout") && timeout < 20*time.Minute {
		timeout = 20 * time.Minute
	}

	return &bootstrap.Options{
		Kubeconfig:         kubeconf,
		PlatformComponents: componentsAbsPath,
		PlatformConfigs:    configsAbsPath,
		Local:              flags.Local,
		WithESCSSPath:      cssAbsPath,
		Credentials:        creds,
		Catalog:            loadedCatalog,
		ClusterConfig:      clusterConfig,
		DryRun:             flags.DryRun,
		Timeout:            timeout,
		ClusterName:        clusterName,
		WorkDir:            cwd,
		ConfigFilePath:     configFilePath,
		Catalogs:           catalogOptions.Catalogs,
		CatalogOverwrite:   catalogOptions.Overwrite,
		BootstrapCatalog:   bootstrapCatalog,
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
			Name:        "local",
			Usage:       "Provision an isolated local evaluation environment. Local testing only; not for production use.",
			Destination: &flags.Local,
		},
		&cli.StringFlag{
			Name:        "with-es-css-file",
			Usage:       "Path to the ClusterSecretStore manifest file (supports go-template + sprig)",
			Destination: &flags.ClusterSecretStorePath,
		},
		&cli.BoolFlag{
			Name:  "with-es-crds",
			Usage: "Deprecated: ignored because CRDs are applied automatically during bootstrap.",
		},
		&cli.BoolFlag{
			Name:  "with-prometheus-crds",
			Usage: "Deprecated: ignored because CRDs are applied automatically during bootstrap.",
		},
		&cli.StringFlag{
			Name:        "platform-components",
			Value:       render.DefaultPlatformComponentsPath,
			Usage:       "Path to the platform-components directory",
			Destination: &flags.PlatformComponentsPath,
		},
		&cli.StringFlag{
			Name:        "platform-configs",
			Value:       render.DefaultPlatformConfigsPath,
			Usage:       "Path to platform-configs directory",
			Destination: &flags.PlatformConfigsPath,
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

func printDeprecatedBootstrapCRDFlagNotice(cmd *cli.Command) {
	for _, flagName := range deprecatedBootstrapCRDFlags {
		if cmd.IsSet(flagName) {
			log.Warn().Msgf("--%s is deprecated and ignored; CRDs are applied automatically during bootstrap.\n", flagName)
		}
	}
}
