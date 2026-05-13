package generate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/render"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
)

type Options struct {
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

// buildTemplateContext creates a map for rendering templates with cluster config, catalog services, and env vars.
func buildTemplateContext(cluster config.Cluster, cat catalog.Catalog, em envconfig.EnvMap) (map[string]any, error) {
	clusterMap, err := toJSONMap(cluster)
	if err != nil {
		return nil, fmt.Errorf("convert cluster config to map: %w", err)
	}

	return map[string]any{
		"env":     em,
		"cluster": clusterMap,
		"catalog": resolveCatalog(cat),
	}, nil
}

func (o *Options) resolveOutputPath(result render.TemplateResult, clusterName string) string {
	trimmedPath := render.StripProviderPath(result.Path)
	trimmedPath = strings.ReplaceAll(trimmedPath, "example", clusterName)
	trimmedPath = strings.TrimSuffix(trimmedPath, ".tplt")
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
			"cluster %q still uses placeholder provider %q; supported providers: %q",
			clusterBlock.Name,
			clusterBlock.Terraform.Provider,
			supportedProviderList(),
		)
	}
	if !render.SupportedProviders[provider] {
		return "", fmt.Errorf("unsupported provider %q for cluster %q; supported providers: %q", provider, clusterBlock.Name, supportedProviderList())
	}
	return provider, nil
}

func resolveCatalog(cat catalog.Catalog) map[string]any {
	if cat.Services == nil {
		return map[string]any{
			"services": map[string]any{},
		}
	}

	services := make(map[string]any, len(cat.Services))
	for serviceName, service := range cat.Services {
		services[serviceName] = map[string]any{
			"status":       service.Spec.Status,
			"chartPath":    service.Spec.ChartPath,
			"clusterTypes": service.Spec.ClusterTypes,
		}
	}

	return map[string]any{
		"services": services,
	}
}

func toJSONMap(value any) (map[string]any, error) {
	rawJSON, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var out map[string]any
	if err := json.Unmarshal(rawJSON, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (o *Options) cleanupOldFiles() error {
	if o.DryRun {
		return nil
	}

	if o.TemplateType == render.All || o.TemplateType == render.Terraform {
		deletePath := filepath.Join(o.ManagedCatalogPath, render.Terraform.String())
		if err := os.RemoveAll(deletePath); err != nil {
			return fmt.Errorf("removing directory %q: %w", deletePath, err)
		}
	}
	if o.TemplateType == render.All || o.TemplateType == render.Helm {
		deletePath := filepath.Join(o.ManagedCatalogPath, render.Helm.String())
		if err := os.RemoveAll(deletePath); err != nil {
			return fmt.Errorf("removing directory %q: %w", deletePath, err)
		}
	}
	return nil
}

func (o *Options) writeTemplateResults(results []render.TemplateResult) error {
	for _, t := range results {
		if o.DryRun {
			fmt.Println("DRY-RUN: " + t.Path)
			continue
		}

		err := os.MkdirAll(filepath.Dir(t.Path), 0750)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create template directory: %w", err)
		}

		err = os.WriteFile(t.Path, []byte(t.Content), 0644)
		if err != nil {
			return fmt.Errorf("write template file: %w", err)
		}
	}
	return nil
}

// processClusters loads config, validates, and generates template results for all clusters.
func (o *Options) processClusters() ([]render.TemplateResult, error) {
	catalogOptions := catalog.LoadOptions{
		CatalogPath: o.CatalogPath,
		Overwrite:   o.CatalogOverwrite,
	}

	cs := config.NewConfigStoreWithCatalog(o.ConfigFilePath, catalogOptions)
	if CnfLoadErr := cs.Load(); CnfLoadErr != nil {
		return nil, fmt.Errorf("load config: %w", CnfLoadErr)
	}

	if err := cs.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	cnf := cs.GetConfig()
	var allResults []render.TemplateResult

	cat, err := cs.GetCatalog()
	if err != nil {
		return nil, fmt.Errorf("load catalog: %w", err)
	}

	dotEnvMap, err := envconfig.GetCurrentDotEnv(o.EnvPath)
	if err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	for _, clusterBlock := range cnf.Clusters {
		tmplContext, err := buildTemplateContext(clusterBlock, cat, dotEnvMap)
		if err != nil {
			return nil, fmt.Errorf("build template context for cluster %q: %w", clusterBlock.Name, err)
		}

		provider := ""
		if o.TemplateType != render.Helm {
			provider, err = resolveProvider(clusterBlock)
			if err != nil {
				return nil, fmt.Errorf("resolve provider for cluster %q: %w", clusterBlock.Name, err)
			}
		}

		clusterTplResults, err := render.TemplateFiles(
			render.TemplateOptions{
				Type:        o.TemplateType,
				Provider:    provider,
				CatalogPath: o.CatalogPath,
				Overwrite:   o.CatalogOverwrite,
				Data:        tmplContext,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("template files: %w", err)
		}

		for i, result := range clusterTplResults {
			if result.Error != nil {
				return nil, fmt.Errorf("template error: %w", result.Error)
			}
			trimmedPath := o.resolveOutputPath(result, clusterBlock.Name)
			clusterTplResults[i].Path = trimmedPath
		}
		allResults = append(allResults, clusterTplResults...)
	}

	return allResults, nil
}

func (o *Options) Run() error {
	allResults, errProcess := o.processClusters()
	if errProcess != nil {
		return errProcess
	}

	if errCleanup := o.cleanupOldFiles(); errCleanup != nil {
		return fmt.Errorf("cleanup old files: %w", errCleanup)
	}

	if errWriteTpls := o.writeTemplateResults(allResults); errWriteTpls != nil {
		return fmt.Errorf("generate files: %w", errWriteTpls)
	}

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
