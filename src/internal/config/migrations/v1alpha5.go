package migrations

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// migrateV1Alpha5Config migrates configurations with version ConfigVersionV1Alpha5 to the ConfigVersionV1Alpha6 schema format.
// It sets argocd.clusterSecrets.provider to "external-secrets" on hub clusters when multiple clusters exist,
// and deletes any legacy argocd key on spoke clusters.
func migrateV1Alpha5Config(config map[string]any) error {
	log.Info().Msg("migrating config from v1alpha5 format to v1alpha6")
	config["version"] = ConfigVersionV1Alpha6

	clusters, ok := config["clusters"].([]any)
	if !ok {
		return nil
	}

	hasMultipleClusters := len(clusters) > 1

	for i, clusterRaw := range clusters {
		cluster, ok := clusterRaw.(map[string]any)
		if !ok {
			continue
		}

		clusterType, _ := cluster["type"].(string)

		if clusterType == "spoke" {
			delete(cluster, "argocd")
			continue
		}

		argocdRaw, hasArgoCD := cluster["argocd"]
		isHub := clusterType == "hub" || (hasArgoCD && argocdRaw != nil)
		if isHub && hasMultipleClusters {
			if !hasArgoCD || argocdRaw == nil {
				continue
			}
			argocd, ok := argocdRaw.(map[string]any)
			if !ok {
				return fmt.Errorf("%s.argocd must be an object", clusterLabel(cluster, i))
			}

			clusterSecretsRaw, hasClusterSecrets := argocd["clusterSecrets"]
			if !hasClusterSecrets || clusterSecretsRaw == nil {
				argocd["clusterSecrets"] = map[string]any{"provider": "external-secrets"}
			} else if csMap, ok := clusterSecretsRaw.(map[string]any); ok {
				if _, hasProvider := csMap["provider"]; !hasProvider {
					csMap["provider"] = "external-secrets"
				}
			}
		}
	}

	return nil
}
