package migrations

import (
	"fmt"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/rs/zerolog/log"
)

// migrateV1Alpha3Config migrates configurations with version ConfigVersionV1Alpha3 to the ConfigVersionV1Alpha4 schema format.
func migrateV1Alpha3Config(config map[string]any) error {
	log.Info().Msg("migrating config from v1alpha3 format to v1alpha4")
	config["version"] = ConfigVersionV1Alpha4

	clustersRaw, ok := config["clusters"]
	if !ok {
		return nil
	}

	clusters, ok := clustersRaw.([]any)
	if !ok {
		return nil
	}

	for i, clusterRaw := range clusters {
		cluster, ok := clusterRaw.(map[string]any)
		if !ok {
			continue
		}

		if err := migrateV1Alpha3Cluster(cluster, i); err != nil {
			return fmt.Errorf("cannot migrate cluster number %d: %w", i, err)
		}

		clusters[i] = cluster
	}

	config["clusters"] = clusters
	return nil
}

func migrateV1Alpha3Cluster(cluster map[string]any, clusterIndex int) error {
	return ensureV1Alpha3GeneralCatalog(cluster, clusterIndex)
}

func ensureV1Alpha3GeneralCatalog(cluster map[string]any, clusterIndex int) error {
	catalogsRaw, exists := cluster["catalogs"]
	if !exists || catalogsRaw == nil {
		cluster["catalogs"] = []any{catalog.DefaultGeneralCatalog}
		return nil
	}
	return nil
}
