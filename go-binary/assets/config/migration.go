package config

import (
	"fmt"
	"reflect"
	"strings"
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
