package migrations

import (
	"fmt"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/service"
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
	if err := ensureV1Alpha3GeneralCatalog(cluster); err != nil {
		return err
	}

	return migrateV1Alpha3ArgoCDSelfManaged(cluster, clusterIndex)
}

func ensureV1Alpha3GeneralCatalog(cluster map[string]any) error {
	catalogsRaw, exists := cluster["catalogs"]
	if !exists || catalogsRaw == nil {
		cluster["catalogs"] = []any{catalog.DefaultGeneralCatalog}
		return nil
	}
	return nil
}

func migrateV1Alpha3ArgoCDSelfManaged(cluster map[string]any, clusterIndex int) error {
	servicesRaw, ok := cluster["services"]
	if !ok {
		return nil
	}

	servicesMap, ok := servicesRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.services must be an object", clusterLabel(cluster, clusterIndex))
	}

	argocdRaw, exists := servicesMap["argocd"]
	if !exists {
		return nil
	}

	argocdService, ok := argocdRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.services.argocd must be an object", clusterLabel(cluster, clusterIndex))
	}

	statusRaw, hasStatus := argocdService["status"]
	if hasStatus && statusRaw != nil {
		status, ok := statusRaw.(string)
		if !ok {
			return fmt.Errorf("%s.services.argocd.status must be a string", clusterLabel(cluster, clusterIndex))
		}
		if status != string(service.StatusEnabled) && status != string(service.StatusDisabled) {
			return fmt.Errorf("%s.services.argocd.status must be either %q or %q", clusterLabel(cluster, clusterIndex), service.StatusEnabled, service.StatusDisabled)
		}

		argocdConfig, err := ensureNestedObject(cluster, "argocd", clusterLabel(cluster, clusterIndex))
		if err != nil {
			return err
		}
		argocdConfig["selfManaged"] = status
	}

	delete(servicesMap, "argocd")
	return nil
}
