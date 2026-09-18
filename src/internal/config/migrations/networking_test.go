package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateIngressNetworking(t *testing.T) {
	for _, tt := range []struct {
		name         string
		cluster      map[string]any
		wantMigrated bool
		wantErr      string
	}{
		{name: "legacy class", cluster: map[string]any{"ingressClassName": "custom"}, wantMigrated: true},
		{name: "omitted class", cluster: map[string]any{}},
		{name: "invalid class", cluster: map[string]any{"ingressClassName": 42}, wantErr: "must be a string"},
		{name: "structured config", cluster: map[string]any{"networking": map[string]any{"type": "ingress"}}},
		{name: "reintroduced field", cluster: map[string]any{"ingressClassName": "custom", "networking": map[string]any{"ingress": map[string]any{"className": "custom"}}}},
		{name: "empty networking is already structured", cluster: map[string]any{"ingressClassName": "custom", "networking": map[string]any{}}},
		{name: "null networking is not legacy", cluster: map[string]any{"ingressClassName": "custom", "networking": nil}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := map[string]any{"version": ConfigVersionV1Alpha4, "clusters": []any{tt.cluster}}
			migrated, err := Apply(t.TempDir(), cfg)
			assert.Equal(t, ConfigVersionV1Alpha4, cfg["version"])
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMigrated, migrated)
			if !tt.wantMigrated {
				return
			}
			assert.NotContains(t, tt.cluster, "ingressClassName")
			networking := tt.cluster["networking"].(map[string]any)
			assert.Equal(t, "ingress", networking["type"])
			assert.Equal(t, "custom", networking["ingress"].(map[string]any)["className"])
			migrated, err = Apply(t.TempDir(), cfg)
			require.NoError(t, err)
			assert.False(t, migrated)
		})
	}
}
