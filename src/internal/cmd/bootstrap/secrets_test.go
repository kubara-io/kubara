package bootstrap

import (
	"testing"

	"github.com/kubara-io/kubara/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateGitRepositorySecret(t *testing.T) {
	sm := &SecretManager{}

	tests := []struct {
		name       string
		cluster    *config.Cluster
		creds      *Credentials
		wantName   string
		wantURL    string
		wantData   map[string]string
		unwantKeys []string
	}{
		{
			name: "creates standard HTTPS repository secret from GITOPS credentials",
			cluster: &config.Cluster{
				Name:  "test",
				Stage: "dev",
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
							Configs: config.Repository{
								URL: "https://github.com/example/repo.git",
							},
						},
					},
				},
			},
			creds: &Credentials{
				GitUsername: "machine-user",
				GitPassword: "token",
			},
			wantName: "https-init-repo-access",
			wantURL:  "https://github.com/example/repo.git",
			wantData: map[string]string{
				"username":           "machine-user",
				"password":           "token",
				"forceHttpBasicAuth": "true",
				"type":               "git",
				"project":            "test-dev",
			},
			unwantKeys: []string{"sshPrivateKey", "githubAppPrivateKey"},
		},
		{
			name: "creates SSH repository secret from GITOPS credentials",
			cluster: &config.Cluster{
				Name:  "test",
				Stage: "dev",
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
							Configs: config.Repository{
								URL: "ssh://git.example.com/example/repo.git",
							},
						},
					},
				},
			},
			creds: &Credentials{
				GitSSHPrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nkey\n-----END OPENSSH PRIVATE KEY-----",
			},
			wantName: "ssh-init-repo-access",
			wantURL:  "ssh://git.example.com/example/repo.git",
			wantData: map[string]string{
				"sshPrivateKey": "-----BEGIN OPENSSH PRIVATE KEY-----\nkey\n-----END OPENSSH PRIVATE KEY-----",
				"type":          "git",
				"project":       "test-dev",
			},
			unwantKeys: []string{"username", "password", "forceHttpBasicAuth"},
		},
		{
			name: "creates GitHub App repository secret from GITOPS credentials",
			cluster: &config.Cluster{
				Name:  "test",
				Stage: "dev",
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
							Configs: config.Repository{
								URL: "https://github.com/example/repo.git",
							},
						},
					},
				},
			},
			creds: &Credentials{
				GitGitHubAppID:                "123",
				GitGitHubAppInstallationID:    "456",
				GitGitHubAppPrivateKey:        "-----BEGIN RSA PRIVATE KEY-----\nkey\n-----END RSA PRIVATE KEY-----",
				GitGitHubAppEnterpriseBaseURL: "https://github.example.com/api/v3",
			},
			wantName: "github-app-init-repo-access",
			wantURL:  "https://github.com/example/repo.git",
			wantData: map[string]string{
				"githubAppID":                "123",
				"githubAppInstallationID":    "456",
				"githubAppPrivateKey":        "-----BEGIN RSA PRIVATE KEY-----\nkey\n-----END RSA PRIVATE KEY-----",
				"githubAppEnterpriseBaseUrl": "https://github.example.com/api/v3",
				"type":                       "git",
				"project":                    "test-dev",
			},
			unwantKeys: []string{"username", "password", "forceHttpBasicAuth", "sshPrivateKey"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := sm.createGitRepositorySecret(tt.cluster, tt.creds)
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
		cluster := &config.Cluster{
			Name:  "test",
			Stage: "dev",
		}
		secret := sm.createHelmRepositorySecret(cluster, &Credentials{})
		assert.Nil(t, secret)
	})

	t.Run("creates secret for classic https helm repo", func(t *testing.T) {
		cluster := &config.Cluster{
			Name:  "test",
			Stage: "dev",
			ArgoCD: config.ArgoCD{
				HelmRepo: &config.HelmRepository{
					URL: "https://charts.example.com",
				},
			},
		}
		secret := sm.createHelmRepositorySecret(cluster, &Credentials{
			HelmRepoUsername: "user",
			HelmRepoPassword: "pass",
		})

		require.NotNil(t, secret)
		assert.Equal(t, "https://charts.example.com", secret.StringData["url"])
		assert.Equal(t, "user", secret.StringData["username"])
		assert.Equal(t, "pass", secret.StringData["password"])
		_, hasEnableOCI := secret.StringData["enableOCI"]
		assert.False(t, hasEnableOCI)
	})

	t.Run("creates secret for OCI helm registry and strips oci scheme", func(t *testing.T) {
		cluster := &config.Cluster{
			Name:  "test",
			Stage: "dev",
			ArgoCD: config.ArgoCD{
				HelmRepo: &config.HelmRepository{
					URL: "oci://registry-1.docker.io/bitnamicharts",
				},
			},
		}
		secret := sm.createHelmRepositorySecret(cluster, &Credentials{})

		require.NotNil(t, secret)
		assert.Equal(t, "registry-1.docker.io/bitnamicharts", secret.StringData["url"])
		assert.Equal(t, "true", secret.StringData["enableOCI"])
	})
}
