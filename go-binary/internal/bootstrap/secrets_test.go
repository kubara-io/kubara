package bootstrap

import (
	"testing"

	"github.com/kubara-io/kubara/internal/envconfig"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateHelmRepositorySecret(t *testing.T) {
	sm := &SecretManager{}

	t.Run("returns nil when helm repo URL is missing", func(t *testing.T) {
		secret := sm.createHelmRepositorySecret(&envconfig.EnvMap{
			ProjectName:       "test",
			ProjectStage:      "dev",
			ArgocdHelmRepoUrl: "",
		})
		assert.Nil(t, secret)
	})

	t.Run("returns nil when helm repo URL is legacy placeholder", func(t *testing.T) {
		secret := sm.createHelmRepositorySecret(&envconfig.EnvMap{
			ProjectName:       "test",
			ProjectStage:      "dev",
			ArgocdHelmRepoUrl: "<...>",
		})
		assert.Nil(t, secret)
	})

	t.Run("creates secret for classic https helm repo", func(t *testing.T) {
		secret := sm.createHelmRepositorySecret(&envconfig.EnvMap{
			ProjectName:            "test",
			ProjectStage:           "dev",
			ArgocdHelmRepoUrl:      "https://charts.example.com",
			ArgocdHelmRepoUsername: "user",
			ArgocdHelmRepoPassword: "pass",
		})

		require.NotNil(t, secret)
		assert.Equal(t, "https://charts.example.com", secret.StringData["url"])
		assert.Equal(t, "user", secret.StringData["username"])
		assert.Equal(t, "pass", secret.StringData["password"])
		_, hasEnableOCI := secret.StringData["enableOCI"]
		assert.False(t, hasEnableOCI)
	})

	t.Run("creates secret for OCI helm registry and strips oci scheme", func(t *testing.T) {
		secret := sm.createHelmRepositorySecret(&envconfig.EnvMap{
			ProjectName:       "test",
			ProjectStage:      "dev",
			ArgocdHelmRepoUrl: "oci://registry-1.docker.io/bitnamicharts",
		})

		require.NotNil(t, secret)
		assert.Equal(t, "registry-1.docker.io/bitnamicharts", secret.StringData["url"])
		assert.Equal(t, "true", secret.StringData["enableOCI"])
	})
}
