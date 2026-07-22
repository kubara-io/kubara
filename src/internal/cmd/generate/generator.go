package generate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/render"
	"github.com/kubara-io/kubara/internal/service"

	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
)

type Options struct {
	TemplateType       render.TemplateType
	DryRun             bool
	CWD                string
	ConfigFilePath     string
	Catalogs           []string
	CatalogOverwrite   bool
	PlatformComponents string
	PlatformConfigs    string
	EnvPath            string
}

type buildContext struct {
	Catalog  catalog.Catalog
	EnvMap   envconfig.EnvMap
	Clusters []config.Cluster
}

// getSpokeClusters returns a list of all spoke Clusters of a given cluster list
func getSpokeClusters(clusters []config.Cluster) ([]map[string]any, error) {
	spokeMaps := make([]map[string]any, 0)
	for _, cluster := range clusters {
		if cluster.Type != config.Spoke {
			continue
		}

		spokeMap, err := toJSONMap(cluster)
		if err != nil {
			return nil, fmt.Errorf("convert spoke %q to map: %w", cluster.Name, err)
		}
		spokeMaps = append(spokeMaps, spokeMap)
	}
	return spokeMaps, nil
}

// buildTemplateContext creates a map for rendering templates for a specific cluster based on the build context provided
// if it is called with a hub cluster, the map contains also the full context of the spokes
func buildTemplateContext(cluster config.Cluster, bctx buildContext) (map[string]any, error) {
	clusterMap, err := toJSONMap(cluster)
	if err != nil {
		return nil, fmt.Errorf("convert cluster config to map: %w", err)
	}
	if cluster.Terraform == nil {
		clusterMap["terraform"] = map[string]any{
			"provider": config.TerraformProviderNone,
		}
	}

	context := map[string]any{
		"env":     bctx.EnvMap,
		"cluster": clusterMap,
		"catalog": resolveCatalog(bctx.Catalog),
	}
	if cluster.Type == config.Hub {
		spokes, err := getSpokeClusters(bctx.Clusters)
		if err != nil {
			return nil, err
		}
		if len(spokes) > 0 {
			context["spokes"] = spokes
		}
	}
	return context, nil
}

func (o *Options) resolveOutputPath(result render.TemplateResult, clusterName string) string {
	trimmedPath := render.StripProviderPath(result.Path)
	trimmedPath = strings.TrimSuffix(trimmedPath, ".tplt")
	trimmedPath = strings.ReplaceAll(trimmedPath, render.DefaultPlatformComponentsPath, o.PlatformComponents)
	trimmedPath = strings.ReplaceAll(trimmedPath, render.DefaultPlatformConfigsPath, fmt.Sprintf("%s/%s", o.PlatformConfigs, clusterName))
	return trimmedPath
}

func supportedProviderList() string {
	supported := config.SupportedTerraformProviders()
	providers := make([]string, 0, len(supported))
	for _, provider := range supported {
		providers = append(providers, string(provider))
	}
	return strings.Join(providers, ", ")
}

func resolveProvider(clusterBlock config.Cluster, requireTerraform bool) (string, error) {
	if clusterBlock.Terraform == nil {
		if requireTerraform {
			return "", fmt.Errorf("cluster %q is missing terraform configuration", clusterBlock.Name)
		}
		return "", nil
	}
	provider := clusterBlock.Terraform.Provider
	if provider == config.TerraformProviderNone {
		if requireTerraform {
			return "", fmt.Errorf("cluster %q has terraform provider %q; configure one of: %q", clusterBlock.Name, provider, supportedProviderList())
		}
		return "", nil
	}
	if provider == "" {
		if requireTerraform {
			return "", fmt.Errorf("cluster %q has a terraform block but no provider specified", clusterBlock.Name)
		}
		return "", nil
	}
	if !provider.IsSupported() {
		return "", fmt.Errorf("unsupported provider %q for cluster %q; supported providers: %q", provider, clusterBlock.Name, supportedProviderList())
	}
	return string(provider), nil
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

	var deletePaths []string
	if o.TemplateType != render.Helm {
		deletePaths = append(deletePaths, filepath.Join(o.PlatformComponents, render.Terraform.String()))
		clusterDirs, err := os.ReadDir(o.PlatformConfigs)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read platform config directories: %w", err)
		}
		for _, clusterDir := range clusterDirs {
			if !clusterDir.IsDir() {
				continue
			}
			deletePaths = append(deletePaths, filepath.Join(o.PlatformConfigs, clusterDir.Name(), render.Terraform.String()))
		}
	}
	if o.TemplateType != render.Terraform {
		deletePaths = append(deletePaths, filepath.Join(o.PlatformComponents, render.Helm.String()))
	}
	for _, deletePath := range deletePaths {
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

func buildChartPathServiceIndex(cat catalog.Catalog) map[string]string {
	index := make(map[string]string, len(cat.Services))
	for serviceName, definition := range cat.Services {
		index[definition.Spec.ChartPath] = serviceName
	}
	return index
}

func serviceNameFromTemplatePath(chartPathServiceIndex map[string]string, path string) string {
	pathParts := strings.Split(filepath.ToSlash(path), "/")
	if len(pathParts) < 4 {
		return ""
	}

	switch {
	case pathParts[0] == render.DefaultPlatformComponentsPath && pathParts[1] == render.Helm.String():
		return chartPathServiceIndex[pathParts[2]]
	case len(pathParts) >= 4 && pathParts[0] == render.DefaultPlatformConfigsPath && pathParts[1] == render.Helm.String():
		return chartPathServiceIndex[pathParts[2]]
	default:
		return ""
	}
}

func buildServiceTemplateFilter(cluster config.Cluster, cat catalog.Catalog) render.TemplatePathPredicate {
	chartPathServiceIndex := buildChartPathServiceIndex(cat)

	return func(path string) bool {
		serviceName := serviceNameFromTemplatePath(chartPathServiceIndex, path)
		if serviceName == "" {
			return true
		}
		if catalog.IsBootstrapService(serviceName) {
			return true
		}

		svc, ok := cluster.Services[serviceName]
		return ok && svc.Status == service.StatusEnabled
	}
}

// processClusters loads config, validates, and generates template results for all clusters.
func (o *Options) processClusters() ([]render.TemplateResult, error) {
	catalogOptions := catalog.LoadOptions{
		CWD:       o.CWD,
		Catalogs:  o.Catalogs,
		Overwrite: o.CatalogOverwrite,
	}

	cs := config.NewConfigStore(o.CWD, o.ConfigFilePath, catalogOptions)
	if CnfLoadErr := cs.Load(); CnfLoadErr != nil {
		return nil, fmt.Errorf("load config: %w", CnfLoadErr)
	}

	cnf := cs.GetConfig()
	var allResults []render.TemplateResult
	resultIndex := make(map[string]int)
	resultCluster := make(map[string]string)

	dotEnvMap, err := envconfig.GetCurrentDotEnv(o.EnvPath)
	if err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	for _, cluster := range cnf.Clusters {
		if cluster.Name != filepath.Base(cluster.Name) || cluster.Name == "." || cluster.Name == ".." {
			return nil, fmt.Errorf("cluster name %q must be a path-safe name", cluster.Name)
		}
		cat, err := cs.GetCatalogForCluster(cluster)
		if err != nil {
			return nil, fmt.Errorf("load catalog for cluster %q: %w", cluster.Name, err)
		}

		tmplContext, err := buildTemplateContext(cluster, buildContext{
			Catalog:  cat,
			EnvMap:   dotEnvMap,
			Clusters: cnf.Clusters,
		})
		if err != nil {
			return nil, fmt.Errorf("build template context for cluster %q: %w", cluster.Name, err)
		}

		provider := ""
		templateType := o.TemplateType
		if o.TemplateType == render.Terraform {
			provider, err = resolveProvider(cluster, true)
			if err != nil {
				return nil, fmt.Errorf("resolve provider for cluster %q: %w", cluster.Name, err)
			}
		}
		if o.TemplateType == render.All {
			provider, err = resolveProvider(cluster, false)
			if err != nil {
				return nil, fmt.Errorf("resolve provider for cluster %q: %w", cluster.Name, err)
			}
		}
		pathPredicate := buildServiceTemplateFilter(cluster, cat)
		if o.TemplateType == render.All && provider == "" {
			enabledPath := pathPredicate
			pathPredicate = func(path string) bool {
				parts := strings.Split(filepath.ToSlash(path), "/")
				if len(parts) > 1 && parts[1] == render.Terraform.String() {
					return false
				}
				return enabledPath(path)
			}
		}

		clusterTplResults, err := render.TemplateFiles(
			render.TemplateOptions{
				Type:           templateType,
				Provider:       provider,
				CatalogOptions: config.CatalogLoadOptions(cnf, cluster, catalogOptions),
				Data:           tmplContext,
				PathPredicate:  pathPredicate,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("template files: %w", err)
		}

		for _, result := range clusterTplResults {
			if result.Error != nil {
				return nil, fmt.Errorf("template error: %w", result.Error)
			}
			result.Path = filepath.Clean(o.resolveOutputPath(result, cluster.Name))
			if index, exists := resultIndex[result.Path]; exists {
				if allResults[index].Content != result.Content {
					return nil, fmt.Errorf(
						"clusters %q and %q generate conflicting content for %q",
						resultCluster[result.Path],
						cluster.Name,
						result.Path,
					)
				}
				continue
			}
			resultIndex[result.Path] = len(allResults)
			resultCluster[result.Path] = cluster.Name
			allResults = append(allResults, result)
		}
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
