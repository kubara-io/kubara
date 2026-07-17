package migrations

import (
	"github.com/rs/zerolog/log"
)

// migrateV1Alpha3Config migrates configurations with version ConfigVersionV1Alpha3 to the ConfigVersionV1Alpha4 schema format.
func migrateV1Alpha3Config(config map[string]any) error {
	log.Info().Msg("migrating config from v1alpha3 format to v1alpha4")
	config["version"] = ConfigVersionV1Alpha4
	return nil
}
