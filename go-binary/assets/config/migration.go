package config

import (
	"fmt"
	"sort"
	"strings"

	"kubara/catalog"
)

// migrateRawConfig performs all config migrations on the raw parsed YAML map
// before it is decoded into the typed Config struct. Migrations are gated by
// the config version: only configs without a version (legacy) are migrated.
// After migration the version is set to v1alpha1.
func migrateRawConfig(raw map[string]any) (bool, error) {
	if raw == nil {
		return false, nil
	}

	version, _ := raw["version"].(string)
	if strings.TrimSpace(version) != "" {
		if version != ConfigVersionV1Alpha1 {
			return false, fmt.Errorf("unsupported config version %q", version)
		}
		return false, nil
	}

	clustersRaw, ok := raw["clusters"]
	if !ok {
		raw["version"] = ConfigVersionV1Alpha1
		return true, nil
	}

	clusters, ok := clustersRaw.([]any)
	if !ok {
		raw["version"] = ConfigVersionV1Alpha1
		return true, nil
	}

	for i := range clusters {
		clusterMap, err := toStringMap(clusters[i])
		if err != nil {
			continue
		}

		servicesRaw, ok := clusterMap["services"]
		if !ok {
			continue
		}

		servicesMap, err := toStringMap(servicesRaw)
		if err != nil {
			continue
		}

		// Step 1: Migrate inline config keys into "config" sub-key per service.
		for serviceName, instanceRaw := range servicesMap {
			instanceMap, err := toStringMap(instanceRaw)
			if err != nil {
				continue
			}
			updatedInstance, changed := migrateInlineServiceConfig(instanceMap)
			if changed {
				servicesMap[serviceName] = updatedInstance
			}
		}

		// Step 2: Normalize service keys (camelCase → kebab-case).
		servicesMap = normalizeRawServiceKeys(servicesMap)

		clusterMap["services"] = servicesMap
		clusters[i] = clusterMap
	}

	raw["clusters"] = clusters
	raw["version"] = ConfigVersionV1Alpha1
	return true, nil
}

// normalizeRawServiceKeys converts legacy camelCase service keys to their
// canonical kebab-case names on the raw map level.
func normalizeRawServiceKeys(services map[string]any) map[string]any {
	if len(services) == 0 {
		return services
	}

	keys := make([]string, 0, len(services))
	for key := range services {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	normalized := make(map[string]any, len(services))
	for _, key := range keys {
		canonical := catalog.CanonicalServiceName(key)
		if canonical == "" {
			continue
		}

		// Canonical key wins when both canonical and legacy alias are present.
		if key != canonical {
			if _, canonicalExists := services[canonical]; canonicalExists {
				continue
			}
		}

		if existing, ok := normalized[canonical]; ok {
			normalized[canonical] = mergeRawServiceInstances(existing, services[key])
			continue
		}
		normalized[canonical] = services[key]
	}

	return normalized
}

// mergeRawServiceInstances merges two raw service instance maps, where the
// override values take precedence over base values.
func mergeRawServiceInstances(baseRaw, overrideRaw any) any {
	base, errBase := toStringMap(baseRaw)
	override, errOverride := toStringMap(overrideRaw)
	if errBase != nil || errOverride != nil {
		return overrideRaw
	}

	for key, val := range override {
		base[key] = val
	}
	return base
}

// migrateInlineServiceConfig moves legacy inline config keys (anything that is
// not "status" or "config") into a "config" sub-key.
func migrateInlineServiceConfig(instance map[string]any) (map[string]any, bool) {
	configRaw, hasConfig := instance["config"]
	var configMap map[string]any
	if hasConfig {
		var err error
		configMap, err = toStringMap(configRaw)
		if err != nil {
			// Non-object config should be surfaced by decode/validation later.
			// In this case we cannot reliably merge legacy inline keys.
			return instance, false
		}
	}

	legacyInline := map[string]any{}
	for key, value := range instance {
		if key == "status" || key == "config" {
			continue
		}
		legacyInline[key] = normalizeValue(value)
	}
	if len(legacyInline) == 0 {
		return instance, false
	}

	if !hasConfig {
		configMap = map[string]any{}
	}
	for key, value := range legacyInline {
		// Explicit config key wins when both exist.
		if _, exists := configMap[key]; exists {
			continue
		}
		configMap[key] = value
	}
	for key := range legacyInline {
		delete(instance, key)
	}
	instance["config"] = configMap

	return instance, true
}
