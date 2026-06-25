package config

import (
	"fmt"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
)

func applyMigrations(config map[string]any) (bool, error) {
	var migrated bool
	if isLegacyConfig(config) {
		if err := migrateLegacyConfig(config); err != nil {
			return false, fmt.Errorf("migrate legacy config: %w", err)
		}
		migrated = true
	}

	if isV1Alpha1Config(config) {
		if err := migrateV1Alpha1Config(config); err != nil {
			return false, fmt.Errorf("migrate V1Alpha1 config: %w", err)
		}
		migrated = true
	}

	return migrated, nil
}

func isLegacyConfig(raw map[string]any) bool {
	_, hasVersion := raw["version"]
	return !hasVersion
}

func isV1Alpha1Config(raw map[string]any) bool {
	version, hasVersion := raw["version"]
	return version == ConfigVersionV1Alpha1 && hasVersion
}

func migrateLegacyConfig(raw map[string]any) error {
	clustersRaw, ok := raw["clusters"]
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

		if err := migrateLegacyCluster(cluster, i); err != nil {
			return fmt.Errorf("cannot migrate cluster number %d: %w", i, err)
		}

		clusters[i] = cluster
	}

	raw["clusters"] = clusters
	return nil
}

func migrateV1Alpha1Config(config map[string]any) error {
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
		return fmt.Errorf("%s.services must be an object", legacyClusterLabel(cluster, clusterIndex))
	}

	metallb, ok := servicesMap["metallb"].(map[string]any)
	if !ok {
		metallb = map[string]any{}
	}

	metallb["config"] = map[string]any{
		"publicLoadBalancerIPs":   publicIps,
		"loadBalancerAddressPool": []any{privateIps},
	}
	servicesMap["metallb"] = metallb
	delete(cluster, "publicLoadBalancerIP")
	delete(cluster, "privateLoadBalancerIP")
	return nil
}

func migrateLegacyCluster(cluster map[string]any, clusterIndex int) error {
	clusterTypeRaw := cluster["type"]
	if clusterTypeRaw != nil {
		clusterType, ok := clusterTypeRaw.(string)
		if !ok {
			return fmt.Errorf("cluster.type must be a string")
		}

		switch clusterType {
		case "worker":
			cluster["type"] = "spoke"
		default:
			cluster["type"] = "hub"
		}
	}

	servicesRaw, ok := cluster["services"]
	if !ok {
		return nil
	}

	servicesMap, ok := servicesRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.services must be an object", legacyClusterLabel(cluster, clusterIndex))
	}

	serviceContext := legacyClusterLabel(cluster, clusterIndex)
	migratedServices := make(map[string]any, len(servicesMap))
	sourceByCanonical := make(map[string]string, len(servicesMap))

	for originalName, serviceRaw := range servicesMap {
		canonicalName := catalog.CanonicalServiceName(originalName)
		if previousName, exists := sourceByCanonical[canonicalName]; exists {
			return fmt.Errorf("%s.services has conflicting keys %q and %q for canonical service %q", serviceContext, previousName, originalName, canonicalName)
		}

		if serviceMap, ok := serviceRaw.(map[string]any); ok {
			if err := migrateLegacyService(canonicalName, serviceMap, serviceContext); err != nil {
				return err
			}
			migratedServices[canonicalName] = serviceMap
		} else {
			migratedServices[canonicalName] = serviceRaw
		}

		sourceByCanonical[canonicalName] = originalName
	}

	cluster["services"] = migratedServices
	return nil
}

func migrateLegacyService(serviceName string, serviceMap map[string]any, clusterContext string) error {
	serviceContext := fmt.Sprintf("%s.services.%s", clusterContext, serviceName)

	if serviceName == "cert-manager" {
		if err := migrateLegacyClusterIssuer(serviceMap, serviceContext); err != nil {
			return err
		}
	}

	switch serviceName {
	case "kube-prometheus-stack", "loki":
		if err := migrateLegacyStorageClassName(serviceMap, serviceContext); err != nil {
			return err
		}
	}

	if err := migrateLegacyIngressAnnotations(serviceMap, serviceContext); err != nil {
		return err
	}

	return nil
}

func migrateLegacyClusterIssuer(serviceMap map[string]any, serviceContext string) error {
	clusterIssuer, ok := serviceMap["clusterIssuer"]
	if !ok {
		return nil
	}

	configMap, err := ensureNestedObject(serviceMap, "config", serviceContext)
	if err != nil {
		return err
	}
	if _, exists := configMap["clusterIssuer"]; exists {
		return fmt.Errorf("%s has both legacy clusterIssuer and config.clusterIssuer", serviceContext)
	}

	configMap["clusterIssuer"] = clusterIssuer
	delete(serviceMap, "clusterIssuer")
	return nil
}

func migrateLegacyStorageClassName(serviceMap map[string]any, serviceContext string) error {
	storageClassName, ok := serviceMap["storageClassName"]
	if !ok {
		return nil
	}

	storageMap, err := ensureNestedObject(serviceMap, "storage", serviceContext)
	if err != nil {
		return err
	}
	if _, exists := storageMap["className"]; exists {
		return fmt.Errorf("%s has both legacy storageClassName and storage.className", serviceContext)
	}

	storageMap["className"] = storageClassName
	delete(serviceMap, "storageClassName")
	return nil
}

func migrateLegacyIngressAnnotations(serviceMap map[string]any, serviceContext string) error {
	ingressRaw, ok := serviceMap["ingress"]
	if !ok {
		return nil
	}

	ingressMap, ok := ingressRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.ingress must be an object", serviceContext)
	}

	annotations, ok := ingressMap["annotations"]
	if !ok {
		return nil
	}

	networkingMap, err := ensureNestedObject(serviceMap, "networking", serviceContext)
	if err != nil {
		return err
	}
	if _, exists := networkingMap["annotations"]; exists {
		return fmt.Errorf("%s has both legacy ingress.annotations and networking.annotations", serviceContext)
	}

	networkingMap["annotations"] = annotations
	delete(ingressMap, "annotations")
	if len(ingressMap) == 0 {
		delete(serviceMap, "ingress")
	} else {
		serviceMap["ingress"] = ingressMap
	}

	return nil
}

func ensureNestedObject(parent map[string]any, key, context string) (map[string]any, error) {
	raw, exists := parent[key]
	if !exists || raw == nil {
		nested := map[string]any{}
		parent[key] = nested
		return nested, nil
	}

	nested, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s.%s must be an object", context, key)
	}

	return nested, nil
}

func legacyClusterLabel(cluster map[string]any, clusterIndex int) string {
	if name, ok := cluster["name"].(string); ok && strings.TrimSpace(name) != "" {
		return fmt.Sprintf("cluster %q", name)
	}

	return fmt.Sprintf("clusters[%d]", clusterIndex)
}
