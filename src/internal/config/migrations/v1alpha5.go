package migrations

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// migrateV1Alpha5Config migrates v1alpha5 to the v1alpha6 schema.
// this includes new networking / gateway-api routing support and
// it sets argocd.clusterSecrets.provider to "external-secrets" on
// hub clusters when multiple clusters exist and deletes any legacy
// argocd key on spoke clusters.
func migrateV1Alpha5Config(config map[string]any) error {
	log.Info().Msg("migrating config from v1alpha5 format to v1alpha6")
	config["version"] = ConfigVersionV1Alpha6

	clusters, ok := config["clusters"].([]any)
	if !ok {
		return nil
	}

	hasMultipleClusters := len(clusters) > 1

	for i, item := range clusters {
		cluster, ok := item.(map[string]any)
		if !ok {
			continue
		}

		if err := migrateGatewayAPISupport(cluster); err != nil {
			return fmt.Errorf("networking config migration didn't work for cluster %q: %w", clusterLabel(cluster, i), err)
		}

		if err := migrateCredentialManagement(cluster, hasMultipleClusters); err != nil {
			return fmt.Errorf("credential management migration didn't work for cluster %q: %w", clusterLabel(cluster, i), err)
		}
	}

	return nil
}

func migrateGatewayAPISupport(cluster map[string]any) error {
	if _, structured := cluster["networking"]; structured {
		return nil
	}
	rawClass, legacy := cluster["ingressClassName"]
	if !legacy {
		return nil
	}
	className, ok := rawClass.(string)
	if !ok {
		return fmt.Errorf("ingressClassName must be a string")
	}
	cluster["networking"] = map[string]any{
		"type":    "ingress",
		"ingress": map[string]any{"className": className},
	}
	delete(cluster, "ingressClassName")

	return nil
}

func migrateCredentialManagement(cluster map[string]any, hasMultipleClusters bool) error {
	clusterType, _ := cluster["type"].(string)

	if clusterType == "spoke" {
		delete(cluster, "argocd")
		return nil
	}

	argocdRaw, hasArgoCD := cluster["argocd"]
	isHub := clusterType == "hub" || (hasArgoCD && argocdRaw != nil)
	if isHub && hasMultipleClusters {
		if !hasArgoCD || argocdRaw == nil {
			return nil
		}
		argocd, ok := argocdRaw.(map[string]any)
		if !ok {
			return fmt.Errorf("argocd must be an object")
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

	return nil
}
