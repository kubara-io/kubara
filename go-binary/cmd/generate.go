package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/render"
	"github.com/kubara-io/kubara/internal/utils"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

type GenerateOptions struct {
	TemplateType       render.TemplateType
	DryRun             bool
	CWD                string
	ConfigFilePath     string
	CatalogPath        string
	CatalogOverwrite   bool
	ManagedCatalogPath string
	OverlayValuesPath  string
	EnvPath            string
}

type GenerateFlags struct {
	Terraform          bool
	Helm               bool
	DryRun             bool
	ManagedCatalogPath string
	OverlayValuesPath  string
}

func NewGenerateFlags() *GenerateFlags {
	return &GenerateFlags{
		Terraform:          false,
		Helm:               false,
		DryRun:             false,
		ManagedCatalogPath: render.DefaultManagedCatalogPath,
		OverlayValuesPath:  render.DefaultOverlayValuesPath,
	}
}

// NewGenerateCmd returns the command with flags added
// TODO implement deep-merge and/or --reset flag
func NewGenerateCmd() *cli.Command {
	flags := NewGenerateFlags()
	cmd := &cli.Command{
		Name:        "generate",
		Usage:       "generates files from embedded templates and the config file; by default for both Helm and Terraform",
		UsageText:   "generate [--terraform|--helm] [--managed-catalog <path> --overlay-values <path>] [--catalog <path> [--catalog-overwrite]] [--dry-run]",
		Description: "generate reads config values and templates the embedded Helm and Terraform files.",
		Action: func(c context.Context, cmd *cli.Command) error {
			o, err := flags.ToOptions(cmd)
			if err != nil {
				return fmt.Errorf("couldn't convert flags to options: %w", err)
			}
			return o.Run()
		},
	}

	flags.AddFlags(cmd)

	return cmd
}

func (flags *GenerateFlags) ToOptions(cmd *cli.Command) (*GenerateOptions, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to get configFilePath: %w", err)
	}
	managedCatalogPath, err := utils.GetFullPath(cmd.String("managed-catalog"), cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to get managed catalog path: %w", err)
	}
	overlayValuesPath, err := utils.GetFullPath(cmd.String("overlay-values"), cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to get overlay values path: %w", err)
	}
	catalogOptions, err := catalogLoadOptionsFromCommand(cmd)
	if err != nil {
		return nil, err
	}
	envPath, err := utils.GetFullPath(cmd.String("env-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to get full envPath: %w", err)
	}

	o := &GenerateOptions{
		TemplateType:       render.All,
		DryRun:             flags.DryRun,
		CWD:                cwd,
		ConfigFilePath:     configFilePath,
		CatalogPath:        catalogOptions.CatalogPath,
		CatalogOverwrite:   catalogOptions.Overwrite,
		ManagedCatalogPath: managedCatalogPath,
		OverlayValuesPath:  overlayValuesPath,
		EnvPath:            envPath,
	}

	if flags.Helm && !flags.Terraform {
		o.TemplateType = render.Helm
	} else if flags.Terraform && !flags.Helm {
		o.TemplateType = render.Terraform
	}

	return o, nil
}

func (flags *GenerateFlags) AddFlags(cmd *cli.Command) {
	generateFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:        "terraform",
			Usage:       "Only generate Terraform files",
			Value:       flags.Terraform,
			Destination: &flags.Terraform,
		},
		&cli.BoolFlag{
			Name:        "helm",
			Usage:       "Only generate Helm files",
			Value:       flags.Helm,
			Destination: &flags.Helm,
		},
		&cli.BoolFlag{
			Name:        "dry-run",
			Usage:       "Preview generation without creating files",
			Value:       flags.DryRun,
			Destination: &flags.DryRun,
		},
		&cli.StringFlag{
			Name:        "managed-catalog",
			Usage:       "Path to the managed catalog directory.",
			Value:       render.DefaultManagedCatalogPath,
			Destination: &flags.ManagedCatalogPath,
		},
		&cli.StringFlag{
			Name:        "overlay-values",
			Usage:       "Path to overlay values directory.",
			Value:       render.DefaultOverlayValuesPath,
			Destination: &flags.OverlayValuesPath,
		},
	}

	cmd.Flags = append(cmd.Flags, generateFlags...)
}

// buildTemplateContext creates a map from a config.Cluster struct.
// It converts the struct to a map using JSON tag names for template variables.
func buildTemplateContext(clusterBlock config.Cluster, em envconfig.EnvMap) (map[string]any, error) {
	// Convert struct to JSON using JSON tags (camelCase)
	clusterJSON, err := json.Marshal(clusterBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cluster to JSON: %w", err)
	}

	// Convert JSON back to map with camelCase keys
	var clusterMap map[string]any
	if err := json.Unmarshal(clusterJSON, &clusterMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal clusterJSON to map: %w", err)
	}

	return map[string]any{
		"cluster": clusterMap,
		"env":     em,
	}, nil
}

func (o *GenerateOptions) resolveOutputPath(result render.TemplateResult, clusterName string) string {
	trimmedPath := render.StripProviderPath(result.Path)
	// 1. Rename 'example' directory to cluster name
	trimmedPath = strings.ReplaceAll(trimmedPath, "example", clusterName)
	// 2. Remove the template extension
	trimmedPath = strings.TrimSuffix(trimmedPath, ".tplt")
	// 3. Replace default catalog paths with configured paths
	trimmedPath = strings.ReplaceAll(trimmedPath, render.DefaultManagedCatalogPath, o.ManagedCatalogPath)
	trimmedPath = strings.ReplaceAll(trimmedPath, render.DefaultOverlayValuesPath, o.OverlayValuesPath)
	return trimmedPath
}

func supportedProviderList() string {
	providers := make([]string, 0, len(render.SupportedProviders))
	for provider := range render.SupportedProviders {
		providers = append(providers, provider)
	}
	sort.Strings(providers)
	return strings.Join(providers, ", ")
}

func resolveProvider(clusterBlock config.Cluster) (string, error) {
	if clusterBlock.Terraform == nil {
		return "", fmt.Errorf("cluster %q is missing terraform configuration", clusterBlock.Name)
	}
	provider := strings.ToLower(strings.TrimSpace(clusterBlock.Terraform.Provider))
	if provider == "" {
		return "", fmt.Errorf("cluster %q has a terraform block but no provider specified", clusterBlock.Name)
	}
	if provider == "<provider>" {
		return "", fmt.Errorf(
			"cluster %q still uses placeholder provider %q; supported providers: %s",
			clusterBlock.Name,
			clusterBlock.Terraform.Provider,
			supportedProviderList(),
		)
	}
	if !render.SupportedProviders[provider] {
		return "", fmt.Errorf("unsupported provider %q for cluster %q; supported providers: %s", provider, clusterBlock.Name, supportedProviderList())
	}
	return provider, nil
}

func (o *GenerateOptions) cleanupOldFiles() error {
	if o.DryRun {
		return nil
	}

	if o.TemplateType == render.All || o.TemplateType == render.Terraform {
		deletePath := filepath.Join(o.ManagedCatalogPath, render.Terraform.String())
		if err := os.RemoveAll(deletePath); err != nil {
			return fmt.Errorf("removing directory %s: %v", deletePath, err)
		}
	}
	if o.TemplateType == render.All || o.TemplateType == render.Helm {
		deletePath := filepath.Join(o.ManagedCatalogPath, render.Helm.String())
		if err := os.RemoveAll(deletePath); err != nil {
			return fmt.Errorf("removing directory %s: %v", deletePath, err)
		}
	}
	return nil
}

func (o *GenerateOptions) writeTemplateResults(results []render.TemplateResult) error {
	for _, t := range results {
		if o.DryRun {
			fmt.Println("DRY-RUN: " + t.Path)
			continue
		}

		// Create directory for each template path
		err := os.MkdirAll(filepath.Dir(t.Path), 0750)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("could not create template directory: %w", err)
		}

		// write out template
		err = os.WriteFile(t.Path, []byte(t.Content), 0644)
		if err != nil {
			return fmt.Errorf("could not write template file: %w", err)
		}
	}
	return nil
}

// processClusters loads config, validates, and generates template results for all clusters.
func (o *GenerateOptions) processClusters() ([]render.TemplateResult, error) {
	cs := config.NewConfigStoreWithCatalog(o.ConfigFilePath, catalog.LoadOptions{
		CatalogPath: o.CatalogPath,
		Overwrite:   o.CatalogOverwrite,
	})
	if CnfLoadErr := cs.Load(); CnfLoadErr != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", o.ConfigFilePath, CnfLoadErr)
	}

	if err := cs.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	cnf := cs.GetConfig()
	var allResults []render.TemplateResult

	dotEnvMap, err := envconfig.GetCurrentDotEnv(o.EnvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load env from envPath:%w", err)
	}

	for _, clusterBlock := range cnf.Clusters {
		tmplContext, err := buildTemplateContext(clusterBlock, dotEnvMap)
		if err != nil {
			return nil, fmt.Errorf("failed to build template context for cluster %s: %w", clusterBlock.Name, err)
		}

		provider := ""
		if o.TemplateType != render.Helm {
			provider, err = resolveProvider(clusterBlock)
			if err != nil {
				return nil, err
			}
		}

		clusterTplResults, err := render.TemplateAllFilesForProvider(
			o.TemplateType,
			tmplContext,
			provider,
		)
		if err != nil {
			return nil, fmt.Errorf("could not template files: %w", err)
		}

		for i, result := range clusterTplResults {
			if result.Error != nil {
				return nil, fmt.Errorf("error in template: %w", result.Error)
			}
			trimmedPath := o.resolveOutputPath(result, clusterBlock.Name)

			clusterTplResults[i].Path = trimmedPath
		}
		allResults = append(allResults, clusterTplResults...)
	}

	return allResults, nil
}

func (o *GenerateOptions) Run() error {

	allResults, errProcess := o.processClusters()
	if errProcess != nil {
		return errProcess
	}

	// Delete old managed-catalog files
	if errCleanup := o.cleanupOldFiles(); errCleanup != nil {
		return fmt.Errorf("cleanup old files: %w", errCleanup)
	}

	// Create all templates
	if errWriteTpls := o.writeTemplateResults(allResults); errWriteTpls != nil {
		return fmt.Errorf("generating files failed: %w", errWriteTpls)
	}

	// TODO improve output
	if o.DryRun {
		log.Info().Msg("DRY-RUN successful.")
		return nil
	}
	log.Info().Msg("All files generated successfully.")
	_, err := color.New(color.FgGreen).Println("✅ Templating complete! Don't forget to PUSH the changes to apply them!")
	if err != nil {
		return err
	}
	return nil
}
