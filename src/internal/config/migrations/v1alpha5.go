package migrations

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// migrateV1Alpha5Config migrates v1alpha5 routing settings to the v1alpha6 schema.
func migrateV1Alpha5Config(config map[string]any) error {
	log.Info().Msg("migrating config from v1alpha5 format to v1alpha6")
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
			return fmt.Errorf("%s.ingressClassName must be a string", clusterLabel(cluster, i))
		}
		cluster["networking"] = map[string]any{
			"type":    "ingress",
			"ingress": map[string]any{"className": className},
		}
		delete(cluster, "ingressClassName")
	}
	config["version"] = ConfigVersionV1Alpha6
	return nil
}
