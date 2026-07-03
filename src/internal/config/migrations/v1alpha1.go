package migrations

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// migrateV1Alpha1Config migrates configurations with version ConfigVersionV1Alpha1 to the ConfigVersionV1Alpha2 schema format.
func migrateV1Alpha1Config(config map[string]any) error {
	log.Info().Msg("migrating config from v1alpha1 format to v1alpha2")
	config["version"] = ConfigVersionV1Alpha2
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

		if err := migrateV1Alpha1Cluster(cluster, i); err != nil {
			return fmt.Errorf("cannot migrate cluster number %d: %w", i, err)
		}

		clusters[i] = cluster
	}

	config["clusters"] = clusters
	return nil
}

func migrateV1Alpha1Cluster(cluster map[string]any, clusterIndex int) error {
	publicIps, hasPublic := cluster["publicLoadBalancerIP"]
	privateIps, hasPrivate := cluster["privateLoadBalancerIP"]
	if !hasPublic && !hasPrivate {
		return nil
	}

	servicesRaw, ok := cluster["services"]
	if !ok {
		return nil
	}

	servicesMap, ok := servicesRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.services must be an object", clusterLabel(cluster, clusterIndex))
	}

	metallb, ok := servicesMap["metallb"].(map[string]any)
	if !ok {
		metallb = map[string]any{}
	}

	metallb["config"] = map[string]any{
		"publicLoadBalancerIPs":   publicIps,
		"loadBalancerAddressPool": []any{fmt.Sprintf("%s/32", privateIps)},
	}
	servicesMap["metallb"] = metallb
	delete(cluster, "publicLoadBalancerIP")
	delete(cluster, "privateLoadBalancerIP")
	return nil
}
