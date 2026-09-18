package migrations

import "fmt"

// migrateIngressNetworking upgrades legacy-only configs without changing their version.
// A networking block marks the structured format; any legacy field beside it is
// left for config validation to reject instead of being migrated again.
func migrateIngressNetworking(config map[string]any) (bool, error) {
	migrated := false
	clusters, _ := config["clusters"].([]any)
	for i, item := range clusters {
		cluster, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if _, structured := cluster["networking"]; structured {
			continue
		}
		rawClass, legacy := cluster["ingressClassName"]
		if !legacy {
			continue
		}
		className, ok := rawClass.(string)
		if !ok {
			return false, fmt.Errorf("%s.ingressClassName must be a string", clusterLabel(cluster, i))
		}
		cluster["networking"] = map[string]any{
			"type":    "ingress",
			"ingress": map[string]any{"className": className},
		}
		delete(cluster, "ingressClassName")
		migrated = true
	}
	return migrated, nil
}
