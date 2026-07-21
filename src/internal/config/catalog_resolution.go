package config

import (
	"fmt"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
)

func effectiveBootstrapCatalog(cfg *Config, options catalog.LoadOptions) string {
	bootstrapCatalog := catalog.DefaultBootstrapCatalog
	if strings.TrimSpace(options.BootstrapCatalog) != "" {
		bootstrapCatalog = options.BootstrapCatalog
	}
	if cfg != nil && cfg.BootstrapCatalog != nil {
		bootstrapCatalog = *cfg.BootstrapCatalog
	}
	return bootstrapCatalog
}

func dedupeCatalogSources(sources ...[]string) []string {
	total := 0
	for _, group := range sources {
		total += len(group)
	}

	ordered := make([]string, 0, total)
	seen := make(map[string]struct{}, total)
	for _, group := range sources {
		for _, source := range group {
			source = strings.TrimSpace(source)
			if source == "" {
				continue
			}
			if _, exists := seen[source]; exists {
				continue
			}
			seen[source] = struct{}{}
			ordered = append(ordered, source)
		}
	}

	return ordered
}

// OrderedCatalogSourcesForCluster returns the bootstrap catalog followed by the
// deduplicated cluster and CLI catalogs in precedence order for template loading.
func OrderedCatalogSourcesForCluster(cfg *Config, cluster Cluster, options catalog.LoadOptions) []string {
	ordered := []string{effectiveBootstrapCatalog(cfg, options)}
	ordered = append(ordered, dedupeCatalogSources(cluster.Catalogs, options.Catalogs)...)
	return ordered
}

// EffectiveCatalogLoadOptionsForCluster returns the catalog loader options for
// one cluster using the shared per-cluster precedence rules.
func EffectiveCatalogLoadOptionsForCluster(cfg *Config, cluster Cluster, options catalog.LoadOptions) catalog.LoadOptions {
	return catalog.LoadOptions{
		BootstrapCatalog: effectiveBootstrapCatalog(cfg, options),
		Catalogs:         dedupeCatalogSources(cluster.Catalogs, options.Catalogs),
		Overwrite:        options.Overwrite,
	}
}

func catalogCacheKey(options catalog.LoadOptions) string {
	parts := []string{
		strings.TrimSpace(options.BootstrapCatalog),
		fmt.Sprintf("%t", options.Overwrite),
	}
	parts = append(parts, options.Catalogs...)
	return strings.Join(parts, "\x00")
}
