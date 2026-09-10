package bootstrap

import (
	"testing"

	"github.com/kubara-io/kubara/internal/envconfig"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateGitRepositorySecret(t *testing.T) {
	sm := &SecretManager{}

	t.Run("creates legacy HTTPS repository secret", func(t *testing.T) {
		secret := sm.createGitRepositorySecret(&envconfig.EnvMap{
			ProjectName:            "test",
			ProjectStage:           "dev",
			ArgocdGitHttpsUrl:      "https://github.com/example/repo.git",
			ArgocdGitPatOrPassword: "token",
			ArgocdGitUsername:      "machine-user",
		})

		require.NotNil(t, secret)
		assert.Equal(t, "https-init-repo-access", secret.Name)
		assert.Equal(t, "https://github.com/example/repo.git", secret.StringData["url"])
		assert.Equal(t, "machine-user", secret.StringData["username"])
		assert.Equal(t, "token", secret.StringData["password"])
		assert.Equal(t, "true", secret.StringData["forceHttpBasicAuth"])
		assert.Equal(t, "git", secret.StringData["type"])
		_, hasSSHKey := secret.StringData["sshPrivateKey"]
		assert.False(t, hasSSHKey)
	})

	t.Run("creates SSH repository secret", func(t *testing.T) {
		secret := sm.createGitRepositorySecret(&envconfig.EnvMap{
			ProjectName:            "test",
			ProjectStage:           "dev",
			ArgocdGitAuthMode:      envconfig.GitAuthModeSSH,
			ArgocdGitUrl:           "git@github.com:example/repo.git",
			ArgocdGitSshPrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nkey\n-----END OPENSSH PRIVATE KEY-----",
		})

		require.NotNil(t, secret)
		assert.Equal(t, "ssh-init-repo-access", secret.Name)
		assert.Equal(t, "git@github.com:example/repo.git", secret.StringData["url"])
		assert.Equal(t, "-----BEGIN OPENSSH PRIVATE KEY-----\nkey\n-----END OPENSSH PRIVATE KEY-----", secret.StringData["sshPrivateKey"])
		_, hasUsername := secret.StringData["username"]
		assert.False(t, hasUsername)
		_, hasPassword := secret.StringData["password"]
		assert.False(t, hasPassword)
	})

	t.Run("creates GitHub App repository secret", func(t *testing.T) {
		secret := sm.createGitRepositorySecret(&envconfig.EnvMap{
			ProjectName:                         "test",
			ProjectStage:                        "dev",
			ArgocdGitAuthMode:                   envconfig.GitAuthModeGitHubApp,
			ArgocdGitUrl:                        "https://github.com/example/repo.git",
			ArgocdGitGithubAppID:                "123",
			ArgocdGitGithubAppInstallationID:    "456",
			ArgocdGitGithubAppPrivateKey:        "-----BEGIN RSA PRIVATE KEY-----\nkey\n-----END RSA PRIVATE KEY-----",
			ArgocdGitGithubAppEnterpriseBaseUrl: "https://github.example.com/api/v3",
		})

		require.NotNil(t, secret)
		assert.Equal(t, "github-app-init-repo-access", secret.Name)
		assert.Equal(t, "https://github.com/example/repo.git", secret.StringData["url"])
		assert.Equal(t, "123", secret.StringData["githubAppID"])
		assert.Equal(t, "456", secret.StringData["githubAppInstallationID"])
		assert.Equal(t, "-----BEGIN RSA PRIVATE KEY-----\nkey\n-----END RSA PRIVATE KEY-----", secret.StringData["githubAppPrivateKey"])
		assert.Equal(t, "https://github.example.com/api/v3", secret.StringData["githubAppEnterpriseBaseUrl"])
		_, hasForceHTTPBasicAuth := secret.StringData["forceHttpBasicAuth"]
		assert.False(t, hasForceHTTPBasicAuth)
	})
}

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
