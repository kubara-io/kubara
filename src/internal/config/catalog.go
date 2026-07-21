package config

import (
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
)

func loadBootstrapCatalog(cfg *Config, options catalog.LoadOptions) string {
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

// CatalogLoadOptions returns the catalog loader options for
// one cluster using the shared per-cluster precedence rules.
func CatalogLoadOptions(cfg *Config, cluster Cluster, options catalog.LoadOptions) catalog.LoadOptions {
	return catalog.LoadOptions{
		CWD:              options.CWD,
		BootstrapCatalog: loadBootstrapCatalog(cfg, options),
		Catalogs:         dedupeCatalogSources(cluster.Catalogs, options.Catalogs),
		Overwrite:        options.Overwrite,
	}
}
