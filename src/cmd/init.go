package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/localmode"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

type InitOptions struct {
	local          bool
	renovate       bool
	cwd            string
	configFilePath string
	catalogOptions catalog.LoadOptions
}

type InitFlags struct {
	LocalFlag            bool
	RenovateFlag         bool
	BootstrapCatalogFlag string
}

func NewInitFlags() *InitFlags {
	return &InitFlags{
		LocalFlag:    false,
		RenovateFlag: true,
	}
}

func NewInitCmd() *cli.Command {
	flags := NewInitFlags()

	cmd := &cli.Command{
		Name:        "init",
		Usage:       "Initialize kubara config for your GitOps repository",
		UsageText:   "kubara init [--local] [--renovate=false] [--bootstrap-catalog PATH_OR_OCI]",
		Description: "Initializes the kubara configuration for your GitOps repository, including catalog options and Renovate support for catalog updates. By default, it creates a config file and, if none exists, a renovate.json file. With --local, it writes a local-evaluation cluster profile in config.yaml.",
		Action: func(c context.Context, cmd *cli.Command) error {
			o, err := flags.ToOptions(cmd)
			if err != nil {
				return err
			}
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
	catalogOptions, err := catalogLoadOptionsFromCommand(cmd, flags.BootstrapCatalogFlag)
	if err != nil {
		return nil, fmt.Errorf("get catalog options: %w", err)
	}

	o := &InitOptions{
		local:          flags.LocalFlag,
		renovate:       flags.RenovateFlag,
		cwd:            cwd,
		configFilePath: configFilePath,
		catalogOptions: catalogOptions,
	}
	return o, nil
}

func (flags *InitFlags) AddFlags(cmd *cli.Command) {
	initFlags := []cli.Flag{
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
		&cli.StringFlag{
			Name:        "bootstrap-catalog",
			Usage:       "Path to the bootstrap catalog directory or an OCI reference in the form oci://registry/repository:x.y.z",
			Destination: &flags.BootstrapCatalogFlag,
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
	}

	cmd.Flags = initFlags
}

func (o *InitOptions) Run() error {
	cs := config.NewConfigStore(o.cwd, o.configFilePath, o.catalogOptions)

	configExists, err := utils.FileExist(cs.GetFilepath())
	if err != nil {
		return err
	}

	clusterName := "test-cluster"
	if configExists {
		log.Warn().Str("path", cs.GetFilepath()).Msg("config file already exists; skipping creation")
	} else {
		newCluster, err := config.NewDefaultClusterWithCatalog(o.cwd, o.catalogOptions)
		if err != nil {
			return fmt.Errorf("create cluster: %w", err)
		}

		if o.local {
			localmode.ApplyClusterProfile(&newCluster, localmode.DefaultDNSName("test-cluster", localmode.DefaultProjectStage))
			newCluster.Name = "test-cluster"
			newCluster.Stage = localmode.DefaultProjectStage
		}
		clusterName = newCluster.Name

		cs.GetConfig().Clusters = []config.Cluster{newCluster}
		o.applyBootstrapCatalogOverride(cs.GetConfig())

		if err := cs.SaveToFile(); err != nil {
			return fmt.Errorf("write config file: %w", err)
		}

		if o.local {
			log.Info().Msgf("Generated local-evaluation config in path: %v", cs.GetFilepath())
		} else {
			log.Info().Msgf("Generated config in path: %v", cs.GetFilepath())
		}
	}

	if err := utils.AddGitignore(o.cwd); err != nil {
		return err
	}

	if err := o.ensureEnvExample(); err != nil {
		return err
	}

	if err := o.ensureRenovateConfig(cs); err != nil {
		return err
	}

	if o.local {
		log.Info().Msgf("Next steps: review config.yaml, then run 'kubara bootstrap --local %s'.", clusterName)
		return nil
	}

	log.Info().Msg("Next steps: review config.yaml, then run 'kubara generate'.")
	return nil
}

func (o *InitOptions) applyBootstrapCatalogOverride(cfg *config.Config) {
	bootstrapCatalog := strings.TrimSpace(o.catalogOptions.BootstrapCatalog)
	if bootstrapCatalog != "" {
		cfg.BootstrapCatalog = &bootstrapCatalog
	}
}

type renovateConfig struct {
	Schema         string                  `json:"$schema"`
	Description    []string                `json:"description"`
	Extends        []string                `json:"extends"`
	IgnorePaths    []string                `json:"ignorePaths"`
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
				log.Warn().Str("path", path).Msg("Renovate config already exists and contains the kubara custom manager; skipping creation")
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
		IgnorePaths: []string{
			"platform-components/**",
			"platform-configs/**/helm/**/values.generated.yaml",
			"platform-configs/**/terraform/infrastructure/terraform.tf",
			"platform-configs/**/terraform/bootstrap-tfstate-backend/main.tf",
		},
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

const envExampleTemplate = `# ✅ These variables are optional credentials for 'kubara bootstrap'
# 💡 You can set them as OS environment variables or copy this file to .env

# Git repository credentials (mutually exclusive: choose HTTPS, SSH, or GitHub App)

# https mode: username + password/PAT (if repository is private)
# GITOPS_USERNAME=""
# GITOPS_PASSWORD=""

# Optional: Initial Argo CD wizard account password (if empty, auto-generated during bootstrap)
# ARGOCD_PASSWORD=""

# ssh mode: private key (inline) or path to private key file
# GITOPS_SSH_PRIVATE_KEY=""
# GITOPS_SSH_PRIVATE_KEY_PATH=""

# github-app mode: GitHub App IDs and private key (inline or path)
# GITOPS_GITHUB_APP_ID=""
# GITOPS_GITHUB_APP_INSTALLATION_ID=""
# GITOPS_GITHUB_APP_PRIVATE_KEY=""
# GITOPS_GITHUB_APP_PRIVATE_KEY_PATH=""
# GITOPS_GITHUB_APP_ENTERPRISE_BASE_URL=""

# Optional: Private Helm/OCI repository credentials
# GITOPS_HELM_REPO_USERNAME=""
# GITOPS_HELM_REPO_PASSWORD=""

# Optional: Container Registry Config (base64-encoded dockerconfigjson)
# DOCKERCONFIG_BASE64=""
`

func (o *InitOptions) ensureEnvExample() error {
	path := filepath.Join(o.cwd, "env.example")
	exists, err := utils.FileExist(path)
	if err != nil {
		return fmt.Errorf("check env.example %q: %w", path, err)
	}
	if exists {
		log.Warn().Str("path", path).Msg("env.example already exists; skipping creation")
		return nil
	}

	if err := os.WriteFile(path, []byte(envExampleTemplate), 0o644); err != nil {
		return fmt.Errorf("write env.example: %w", err)
	}

	log.Info().Msgf("Generated env.example in path: %s", path)
	return nil
}
