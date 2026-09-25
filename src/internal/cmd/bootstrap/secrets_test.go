package bootstrap

import (
	"testing"

	"github.com/kubara-io/kubara/internal/envconfig"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateGitRepositorySecret(t *testing.T) {
	sm := &SecretManager{}

	tests := []struct {
		name       string
		env        *envconfig.EnvMap
		wantName   string
		wantURL    string
		wantData   map[string]string
		unwantKeys []string
	}{
		{
			name: "creates legacy HTTPS repository secret",
			env: &envconfig.EnvMap{
				ProjectName:            "test",
				ProjectStage:           "dev",
				ArgocdGitHttpsUrl:      "https://github.com/example/repo.git",
				ArgocdGitPatOrPassword: "token",
				ArgocdGitUsername:      "machine-user",
			},
			wantName: "https-init-repo-access",
			wantURL:  "https://github.com/example/repo.git",
			wantData: map[string]string{
				"username":           "machine-user",
				"password":           "token",
				"forceHttpBasicAuth": "true",
				"type":               "git",
			},
			unwantKeys: []string{"sshPrivateKey", "githubAppPrivateKey"},
		},
		{
			name: "creates standard HTTPS repository secret",
			env: &envconfig.EnvMap{
				ProjectName:            "test",
				ProjectStage:           "dev",
				ArgocdGitAuthMode:      envconfig.GitAuthModeHTTPS,
				ArgocdGitUrl:           "https://github.com/example/repo.git",
				ArgocdGitPatOrPassword: "token",
				ArgocdGitUsername:      "machine-user",
			},
			wantName: "https-init-repo-access",
			wantURL:  "https://github.com/example/repo.git",
			wantData: map[string]string{
				"username":           "machine-user",
				"password":           "token",
				"forceHttpBasicAuth": "true",
				"type":               "git",
			},
			unwantKeys: []string{"sshPrivateKey", "githubAppPrivateKey"},
		},
		{
			name: "creates SSH repository secret",
			env: &envconfig.EnvMap{
				ProjectName:            "test",
				ProjectStage:           "dev",
				ArgocdGitAuthMode:      envconfig.GitAuthModeSSH,
				ArgocdGitUrl:           "[EMAIL_REDACTED]:example/repo.git",
				ArgocdGitSshPrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nkey\n-----END OPENSSH PRIVATE KEY-----",
			},
			wantName: "ssh-init-repo-access",
			wantURL:  "[EMAIL_REDACTED]:example/repo.git",
			wantData: map[string]string{
				"sshPrivateKey": "-----BEGIN OPENSSH PRIVATE KEY-----\nkey\n-----END OPENSSH PRIVATE KEY-----",
				"type":          "git",
			},
			unwantKeys: []string{"username", "password", "forceHttpBasicAuth"},
		},
		{
			name: "creates GitHub App repository secret",
			env: &envconfig.EnvMap{
				ProjectName:                         "test",
				ProjectStage:                        "dev",
				ArgocdGitAuthMode:                   envconfig.GitAuthModeGitHubApp,
				ArgocdGitUrl:                        "https://github.com/example/repo.git",
				ArgocdGitGithubAppID:                "123",
				ArgocdGitGithubAppInstallationID:    "456",
				ArgocdGitGithubAppPrivateKey:        "-----BEGIN RSA PRIVATE KEY-----\nkey\n-----END RSA PRIVATE KEY-----",
				ArgocdGitGithubAppEnterpriseBaseUrl: "https://github.example.com/api/v3",
			},
			wantName: "github-app-init-repo-access",
			wantURL:  "https://github.com/example/repo.git",
			wantData: map[string]string{
				"githubAppID":                "123",
				"githubAppInstallationID":    "456",
				"githubAppPrivateKey":        "-----BEGIN RSA PRIVATE KEY-----\nkey\n-----END RSA PRIVATE KEY-----",
				"githubAppEnterpriseBaseUrl": "https://github.example.com/api/v3",
				"type":                       "git",
			},
			unwantKeys: []string{"username", "password", "forceHttpBasicAuth", "sshPrivateKey"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := sm.createGitRepositorySecret(tt.env)
			require.NotNil(t, secret)
			assert.Equal(t, tt.wantName, secret.Name)
			assert.Equal(t, tt.wantURL, secret.StringData["url"])
			assert.Equal(t, tt.wantName, secret.StringData["name"])
			for k, v := range tt.wantData {
				assert.Equal(t, v, secret.StringData[k])
			}
			for _, k := range tt.unwantKeys {
				_, exists := secret.StringData[k]
				assert.False(t, exists, "key %s should not exist in StringData", k)
			}
		})
	}
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
