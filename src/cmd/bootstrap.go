package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kubara-io/kubara/internal/cmd/bootstrap"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/localmode"
	"github.com/kubara-io/kubara/internal/render"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/urfave/cli/v3"
)

type BootstrapFlags struct {
	WithES                 bool
	WithProm               bool
	Local                  bool
	ClusterSecretStorePath string
	PlatformComponentsPath string
	PlatformConfigsPath    string
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
		Name:        "bootstrap",
		Usage:       "Bootstrap Argo CD onto a cluster",
		UsageText:   "kubara bootstrap CLUSTER_NAME [--local]",
		ArgsUsage:   "CLUSTER_NAME",
		Description: "Bootstraps Argo CD onto the specified cluster and can also install external-secrets and kube-prometheus-stack CRDs. The optional --local mode provisions an isolated local evaluation environment and is not intended for production use.",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:      "cluster-name",
				UsageText: "The cluster name defined in the config",
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

	catalogOptions, err := catalogLoadOptionsFromCommand(cmd)
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

	es := envconfig.NewEnvStore(envFilePath, ".", flags.EnvPrefixFlag)
	if err := es.Load(); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	envMap, err := prepareBootstrapEnv(clusterConfig, es.GetConfig(), flags.Local)
	if err != nil {
		return nil, fmt.Errorf("prepare env: %w", err)
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

	timeout := flags.Timeout
	if flags.Local && !cmd.IsSet("timeout") && timeout < 20*time.Minute {
		timeout = 20 * time.Minute
	}

	return &bootstrap.Options{
		Kubeconfig:         kubeconf,
		PlatformComponents: componentsAbsPath,
		PlatformConfigs:    configsAbsPath,
		WithES:             flags.WithES,
		WithProm:           flags.WithProm,
		Local:              flags.Local,
		WithESCSSPath:      cssAbsPath,
		EnvMap:             envMap,
		Catalog:            catalog,
		ClusterConfig:      clusterConfig,
		DryRun:             flags.DryRun,
		Timeout:            timeout,
		ClusterName:        clusterName,
		WorkDir:            cwd,
		ConfigFilePath:     configFilePath,
		CatalogPath:        catalogOptions.CatalogPath,
		CatalogOverwrite:   catalogOptions.Overwrite,
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

func prepareBootstrapEnv(cluster *config.Cluster, envMap *envconfig.EnvMap, local bool) (*envconfig.EnvMap, error) {
	if err := envMap.Validate(); err != nil {
		return nil, err
	}

	if (envconfig.IsConfiguredEnvValue(envMap.ArgocdGitUsername) && !envconfig.IsConfiguredEnvValue(envMap.ArgocdGitPatOrPassword)) ||
		(!envconfig.IsConfiguredEnvValue(envMap.ArgocdGitUsername) && envconfig.IsConfiguredEnvValue(envMap.ArgocdGitPatOrPassword)) {
		return nil, fmt.Errorf("if you are using a private repository you need to configure both ARGOCD_GIT_PAT_OR_PASSWORD and ARGOCD_GIT_USERNAME")
	}

	if !local {
		return envMap, nil
	}

	prepared := *envMap
	if !envconfig.IsConfiguredEnvValue(prepared.ProjectName) {
		prepared.ProjectName = cluster.Name
	}
	if !envconfig.IsConfiguredEnvValue(prepared.ProjectStage) {
		prepared.ProjectStage = cluster.Stage
	}
	if !envconfig.IsConfiguredEnvValue(prepared.DomainName) {
		prepared.DomainName = localmode.DomainName
	}

	return &prepared, nil
}
