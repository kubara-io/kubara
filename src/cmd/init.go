package cmd

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/localmode"
	"github.com/kubara-io/kubara/internal/utils"
	"github.com/kubara-io/kubara/internal/workflow"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

//go:embed *-catalog-update.yaml kubara-catalog-update.sh install-oras.sh
var catalogUpdateTemplates embed.FS

type InitOptions struct {
	copyPrepFolder bool
	force          bool
	local          bool
	renovate       bool
	githubActions  bool
	gitlabCI       bool
	cwd            string
	configFilePath string
	dotEnvFilePath string
	envVarPrefix   string
	catalogOptions catalog.LoadOptions
}

type InitFlags struct {
	PrepFlag             bool
	ForceFlag            bool
	LocalFlag            bool
	RenovateFlag         bool
	GitHubActionsFlag    bool
	GitLabCIFlag         bool
	EnvFileFlag          string
	EnvPrefixFlag        string
	BootstrapCatalogFlag string
}

func NewInitFlags() *InitFlags {
	return &InitFlags{
		PrepFlag:      false,
		ForceFlag:     false,
		LocalFlag:     false,
		RenovateFlag:  true,
		EnvFileFlag:   ".env",
		EnvPrefixFlag: "KUBARA_",
	}
}

func NewInitCmd() *cli.Command {
	flags := NewInitFlags()

	cmd := &cli.Command{
		Name:        "init",
		Usage:       "Initialize kubara config for your GitOps repository",
		UsageText:   "kubara init [--prep] [--local] [--renovate=false] [--github-actions] [--gitlab-ci] [--bootstrap-catalog PATH_OR_OCI]",
		Description: "Initializes the kubara configuration for your GitOps repository, including environment variables, catalog options, and Renovate support for catalog updates. By default, it creates a config file and, if none exists, a renovate.json file. With --prep, it only generates the .env template for manual configuration. Combined with --local, --prep pre-fills local-evaluation defaults in .env and init writes a local-only cluster profile in config.yaml.",
		Action: func(c context.Context, cmd *cli.Command) error {
			o, _ := flags.ToOptions(cmd)
			return o.Run()
		},
	}

	flags.AddFlags(cmd)

	return cmd
}

func (flags *InitFlags) ToOptions(cmd *cli.Command) (*InitOptions, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get config file path: %w", err)
	}
	dotEnvFilePath, err := utils.GetFullPath(cmd.String("env-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get env file path: %w", err)
	}
	catalogOptions, err := catalogLoadOptionsFromCommand(cmd, flags.BootstrapCatalogFlag)
	if err != nil {
		return nil, fmt.Errorf("get catalog options: %w", err)
	}

	o := &InitOptions{
		copyPrepFolder: flags.PrepFlag,
		force:          flags.ForceFlag,
		local:          flags.LocalFlag,
		renovate:       flags.RenovateFlag,
		githubActions:  flags.GitHubActionsFlag,
		gitlabCI:       flags.GitLabCIFlag,
		cwd:            cwd,
		configFilePath: configFilePath,
		dotEnvFilePath: dotEnvFilePath,
		envVarPrefix:   flags.EnvPrefixFlag,
		catalogOptions: catalogOptions,
	}
	return o, nil
}

func (flags *InitFlags) AddFlags(cmd *cli.Command) {
	initFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:        "prep",
			Value:       flags.PrepFlag,
			Usage:       "Copy embedded prep/ folder into current working directory",
			Destination: &flags.PrepFlag,
		},
		&cli.BoolFlag{
			Name:        "overwrite",
			Value:       flags.ForceFlag,
			Usage:       "Overwrite config if exists",
			Destination: &flags.ForceFlag,
		},
		&cli.BoolFlag{
			Name:        "local",
			Value:       flags.LocalFlag,
			Usage:       "Initialize files for the local evaluation workflow. Local testing only; not for production use.",
			Destination: &flags.LocalFlag,
		},
		&cli.BoolFlag{
			Name:        "renovate",
			Value:       flags.RenovateFlag,
			Usage:       "Generate a Renovate configuration for kubara catalog updates if none exists",
			Destination: &flags.RenovateFlag,
		},
		&cli.BoolFlag{
			Name:        "github-actions",
			Usage:       "Generate a GitHub Actions catalog update workflow if none exists",
			Destination: &flags.GitHubActionsFlag,
		},
		&cli.BoolFlag{
			Name:        "gitlab-ci",
			Usage:       "Generate a GitLab catalog update pipeline if none exists",
			Destination: &flags.GitLabCIFlag,
		},
		&cli.StringFlag{
			Name:        "bootstrap-catalog",
			Usage:       "Path to the bootstrap catalog directory or an OCI reference in the form oci://registry/repository:x.y.z",
			Destination: &flags.BootstrapCatalogFlag,
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
		&cli.StringFlag{
			Name:        "envVarPrefix",
			Value:       flags.EnvPrefixFlag,
			Usage:       "Prefix for envs read from envVars",
			Destination: &flags.EnvPrefixFlag,
		},
	}

	cmd.Flags = initFlags
}

func (o *InitOptions) Run() error {
	es := envconfig.NewEnvStore(o.dotEnvFilePath, ".", o.envVarPrefix)
	cs := config.NewConfigStore(o.cwd, o.configFilePath, o.catalogLoadOptions())

	envLoadErr := es.Load()
	configLoadErr := cs.Load()
	var envValidateErr error
	if o.local {
		if o.copyPrepFolder {
			localmode.PopulateInitEnv(es.GetConfig())
		}
		envValidateErr = es.Validate()
	} else {
		envValidateErr = es.Validate()
		es.SetDefaults()
	}

	if envLoadErr != nil {
		log.Error().Msgf("Reading Env failed. %s", envLoadErr)
		return envLoadErr
	}

	if o.copyPrepFolder {
		return o.runPrepMode(es)
	}

	var err error
	if o.force {
		err = o.runForceMode(es, cs, envValidateErr, configLoadErr)
	} else {
		err = o.runNormalMode(es, cs, envValidateErr)
	}
	if err != nil {
		return err
	}
	return o.ensureCatalogUpdateCI()
}

func (o *InitOptions) catalogLoadOptions() catalog.LoadOptions {
	return o.catalogOptions
}

func (o *InitOptions) applyBootstrapCatalogOverride(cfg *config.Config) {
	bootstrapCatalog := strings.TrimSpace(o.catalogOptions.BootstrapCatalog)
	if bootstrapCatalog != "" {
		cfg.BootstrapCatalog = &bootstrapCatalog
	}
}

func (o *InitOptions) ensureLocalDotEnv(es *envconfig.EnvStore) error {
	if err := utils.AddGitignore(o.cwd); err != nil {
		return err
	}

	_, err := os.Stat(o.dotEnvFilePath)
	if err == nil {
		log.Info().Msgf("Skipping dotenv creation. File exist: %v", es.GetFilepath())
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	content, err := es.GenerateEnvFileFromCurrentValues()
	if err != nil {
		return err
	}
	if err := os.WriteFile(o.dotEnvFilePath, content, 0o600); err != nil {
		return err
	}

	log.Info().Msgf("Generated local-evaluation dotenv in path: %v", es.GetFilepath())
	return nil
}

func (o *InitOptions) runPrepMode(es *envconfig.EnvStore) error {
	if o.local {
		if err := o.ensureLocalDotEnv(es); err != nil {
			return err
		}
		log.Info().Msg("Next steps: update your .env file, then run 'kubara init --local'.")
		return nil
	}

	if err := utils.AddGitignore(o.cwd); err != nil {
		return err
	}

	_, err := os.Stat(o.dotEnvFilePath)
	if err == nil {
		log.Info().Msgf("Skipping dotenv creation. File exist: %v", es.GetFilepath())
		log.Info().Msg("Next steps: fill in your .env file, then run 'kubara init'.")
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	initialEnvs, err := es.GenerateInitialEnvs()
	if err != nil {
		return err
	}
	if err := os.WriteFile(o.dotEnvFilePath, initialEnvs, 0600); err != nil {
		return err
	}

	log.Info().Msgf("Generated .env in path: %s", es.GetFilepath())
	log.Info().Msg("Next steps: fill in your .env file, then run 'kubara init'.")
	return nil
}

func (o *InitOptions) runForceMode(es *envconfig.EnvStore, cs *config.ConfigStore, envValidateErr, configLoadErr error) error {
	if envValidateErr != nil {
		return fmt.Errorf("validate env: %w", envValidateErr)
	}

	fileExists, _ := utils.FileExist(cs.GetFilepath())
	if !fileExists {
		return fmt.Errorf("load config file: %w", configLoadErr)
	}

	if err := workflow.CreateOrUpdateCluster(cs.GetConfig(), es.GetConfig(), o.catalogLoadOptions()); err != nil {
		return fmt.Errorf("create or update cluster from env: %w", err)
	}
	if o.local {
		clusterName := es.GetConfig().ProjectName
		dnsName := localmode.DefaultDNSName(es.GetConfig().ProjectName, es.GetConfig().ProjectStage)
		for i := range cs.GetConfig().Clusters {
			if cs.GetConfig().Clusters[i].Name == clusterName {
				localmode.ApplyClusterProfile(&cs.GetConfig().Clusters[i], dnsName)
				break
			}
		}
	}
	if err := cs.SaveToFile(); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	if err := o.ensureRenovateConfig(cs); err != nil {
		return err
	}

	if o.local {
		log.Info().Msgf("Overwrote local-evaluation config file: %s", cs.GetFilepath())
		log.Info().Msg("Initialized local evaluation workflow successfully")
		log.Info().Msg("Next steps: review config.yaml, then run 'kubara bootstrap --local test-cluster'.")
		return nil
	}

	log.Info().Msgf("overwritten config file: %s", cs.GetFilepath())
	log.Info().Msg("Initialized successfully")
	log.Info().Msg("Next steps: review config.yaml, then run 'kubara generate'.")
	return nil
}

func (o *InitOptions) runNormalMode(es *envconfig.EnvStore, cs *config.ConfigStore, envValidateErr error) error {
	fileExists, err := utils.FileExist(cs.GetFilepath())
	if err != nil {
		return err
	}

	if fileExists {
		if err := o.ensureRenovateConfig(cs); err != nil {
			return err
		}
		log.Info().Msgf("Config file already exist. To overwrite existing variables in the config from env: set flag \"--overwrite\"")
		log.Info().Msg("Initialized successfully")
		if o.local {
			log.Info().Msg("Next steps: review config.yaml, then run 'kubara bootstrap --local test-cluster'.")
		} else {
			log.Info().Msg("Next steps: review config.yaml, then run 'kubara generate'.")
		}
		return nil
	}

	if envValidateErr != nil {
		log.Info().Msgf("Env validation error. If you want to generate an initial .env file, pass the \"--prep\" flag.")
		return fmt.Errorf("validate env: %w", envValidateErr)
	}

	newCluster, err := config.NewClusterFromEnvWithCatalog(es.GetConfig(), o.catalogLoadOptions())
	if err != nil {
		return fmt.Errorf("create cluster from env: %w", err)
	}
	if o.local {
		localmode.ApplyClusterProfile(&newCluster, localmode.DefaultDNSName(es.GetConfig().ProjectName, es.GetConfig().ProjectStage))
	}

	cs.GetConfig().Clusters = []config.Cluster{newCluster}
	o.applyBootstrapCatalogOverride(cs.GetConfig())
	if err := cs.SaveToFile(); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	if err := o.ensureRenovateConfig(cs); err != nil {
		return err
	}

	if o.local {
		log.Info().Msgf("Generated local-evaluation config in path: %v", cs.GetFilepath())
		log.Info().Msgf("Next steps: review config.yaml, then run 'kubara bootstrap --local %s'.", newCluster.Name)
		return nil
	}

	log.Info().Msgf("Generated config in path: %v", cs.GetFilepath())
	log.Info().Msg("Next steps: review config.yaml, then run 'kubara generate'.")
	return nil
}

type renovateConfig struct {
	Schema         string                  `json:"$schema"`
	Description    []string                `json:"description"`
	Extends        []string                `json:"extends"`
	CustomManagers []renovateCustomManager `json:"customManagers"`
}

const kubaraRenovateManagerDescription = "Update kubara catalog OCI references"

type renovateCustomManager struct {
	CustomType          string   `json:"customType"`
	Description         string   `json:"description"`
	ManagerFilePatterns []string `json:"managerFilePatterns"`
	MatchStrings        []string `json:"matchStrings"`
	DatasourceTemplate  string   `json:"datasourceTemplate"`
	VersioningTemplate  string   `json:"versioningTemplate"`
}

func (o *InitOptions) ensureRenovateConfig(cs *config.ConfigStore) error {
	if !o.renovate {
		return nil
	}

	// https://docs.renovatebot.com/configuration-options/#locations-for-configuration-filenames
	renovateFiles := []string{
		"renovate.json",
		"renovate.jsonc",
		"renovate.json5",
		".github/renovate.json",
		".github/renovate.jsonc",
		".github/renovate.json5",
		".gitlab/renovate.json",
		".gitlab/renovate.jsonc",
		".gitlab/renovate.json5",
		".renovaterc",
		".renovaterc.json",
		".renovaterc.jsonc",
		".renovaterc.json5",
	}

	for _, file := range renovateFiles {
		path := filepath.Join(o.cwd, file)
		exists, err := utils.FileExist(path)
		if err != nil {
			return fmt.Errorf("check renovate config %q: %w", path, err)
		}
		if exists {
			containsKubaraManager, err := renovateConfigContainsKubaraManager(path)
			if err != nil {
				return err
			}
			if containsKubaraManager {
				log.Info().Str("path", path).Msg("Renovate config already contains the kubara custom manager")
				return nil
			}
			log.Warn().Str("path", path).Msg("Renovate config already exists; add the kubara custom manager manually if needed")
			return nil
		}
	}

	relativeConfigPath, err := filepath.Rel(o.cwd, cs.GetFilepath())
	if err != nil {
		return fmt.Errorf("get config path relative to working directory: %w", err)
	}
	if relativeConfigPath == ".." || strings.HasPrefix(relativeConfigPath, ".."+string(filepath.Separator)) {
		log.Warn().Str("configPath", cs.GetFilepath()).Str("workDir", o.cwd).Msg("Skipping Renovate config because the kubara config is outside the working directory")
		return nil
	}
	fileMatchPattern := regexp.QuoteMeta(filepath.ToSlash(relativeConfigPath))
	fileMatchPattern = strings.ReplaceAll(fileMatchPattern, "/", `\/`)

	generatedConfig := renovateConfig{
		Schema: "https://docs.renovatebot.com/renovate-schema.json",
		Description: []string{
			"Generated by kubara init.",
			"This configuration adds support for updating kubara catalog OCI references.",
		},
		Extends: []string{"config:recommended"},
		CustomManagers: []renovateCustomManager{
			{
				CustomType:          "regex",
				Description:         kubaraRenovateManagerDescription,
				ManagerFilePatterns: []string{"/^" + fileMatchPattern + "$/"},
				MatchStrings:        []string{`oci:\/\/(?<depName>[^\s"'@]+):(?<currentValue>[^\s/:"'@]+)(?:$|[\s"'#,}\]])`},
				DatasourceTemplate:  "docker",
				VersioningTemplate:  `regex:^(?<major>0|[1-9]\d*)\.(?<minor>0|[1-9]\d*)\.(?<patch>0|[1-9]\d*)$`,
			},
		},
	}
	var content bytes.Buffer
	encoder := json.NewEncoder(&content)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(generatedConfig); err != nil {
		return fmt.Errorf("marshal renovate config: %w", err)
	}

	renovatePath := filepath.Join(o.cwd, "renovate.json")
	if err := os.WriteFile(renovatePath, content.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write renovate config: %w", err)
	}

	log.Info().Msgf("Generated renovate config in path: %s", renovatePath)
	return nil
}

func renovateConfigContainsKubaraManager(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read renovate config %q: %w", path, err)
	}
	return bytes.Contains(content, []byte(kubaraRenovateManagerDescription)), nil
}

func (o *InitOptions) ensureCatalogUpdateCI() error {
	for _, file := range []struct {
		enabled bool
		source  string
		target  string
	}{
		{o.githubActions, "github-catalog-update.yaml", ".github/workflows/kubara-update.yaml"},
		{o.gitlabCI, "gitlab-catalog-update.yaml", ".gitlab-ci.yml"},
		{o.githubActions || o.gitlabCI, "kubara-catalog-update.sh", ".scripts/kubara-catalog-update.sh"},
		{o.gitlabCI, "install-oras.sh", ".scripts/install-oras.sh"},
	} {
		if !file.enabled {
			continue
		}
		path := filepath.Join(o.cwd, file.target)
		if _, err := os.Lstat(path); err == nil {
			log.Info().Msgf("Skipping existing CI file: %s", path)
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("check CI file %q: %w", path, err)
		}
		contents, err := catalogUpdateTemplates.ReadFile(file.source)
		if err != nil {
			return fmt.Errorf("read CI template: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create CI directory: %w", err)
		}
		if err := os.WriteFile(path, contents, 0o644); err != nil {
			return fmt.Errorf("write CI file: %w", err)
		}
		log.Info().Msgf("Generated CI file: %s", path)
	}
	return nil
}
