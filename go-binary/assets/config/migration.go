package config

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"kubara/catalog"
)

func migrateConfig(cfg *Config) (bool, error) {
	if cfg == nil {
		return false, nil
	}

	version := strings.TrimSpace(cfg.Version)
	switch version {
	case "":
		migrated := false
		for i := range cfg.Clusters {
			before := cfg.Clusters[i].Services
			after := normalizeServiceKeys(before)
			if !reflect.DeepEqual(before, after) {
				migrated = true
			}
			cfg.Clusters[i].Services = after
		}

		cfg.Version = ConfigVersionV1Alpha1
		migrated = true
		return migrated, nil
	case ConfigVersionV1Alpha1:
		return false, nil
	default:
		return false, fmt.Errorf("unsupported config version %q", cfg.Version)
	}
}

func normalizeServiceKeys(input Services) Services {
	if input == nil {
		return nil
	}

	normalized := make(Services, len(input))
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		canonical := catalog.CanonicalServiceName(key)
		if canonical == "" {
			continue
		}

		// Canonical key wins when both canonical and legacy alias are present.
		if key != canonical {
			if _, canonicalExists := input[canonical]; canonicalExists {
				continue
			}
		}

		instance := input[key]
		if existing, ok := normalized[canonical]; ok {
			normalized[canonical] = mergeServiceInstances(existing, instance)
			continue
		}
		normalized[canonical] = instance
	}

	return normalized
}

func mergeServiceInstances(base, override ServiceInstance) ServiceInstance {
	out := cloneServiceInstance(base)
	if override.Status != "" {
		out.Status = override.Status
	}
	out.Config = mergeConfigDefaults(out.Config, override.Config)
	return out
}
