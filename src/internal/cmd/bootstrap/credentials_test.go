package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kubara-io/kubara/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCredentials_Precedence(t *testing.T) {
	tempDir := t.TempDir()

	envContent := `
GITOPS_USERNAME=file-user
GITOPS_PASSWORD=file-pass
GITOPS_HELM_REPO_USERNAME=file-helm-user
ARGOCD_HELM_REPO_PASSWORD=file-helm-legacy-pass
`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".env"), []byte(envContent), 0600))

	// OS env overrides file
	t.Setenv("GITOPS_USERNAME", "os-user")
	// OS legacy overrides file canonical
	t.Setenv("ARGOCD_GIT_PASSWORD", "os-legacy-pass")

	creds, err := LoadCredentials(tempDir)
	require.NoError(t, err)

	assert.Equal(t, "os-user", creds.GitUsername)
	assert.Equal(t, "os-legacy-pass", creds.GitPassword)
	assert.Equal(t, "file-helm-user", creds.HelmRepoUsername)
	assert.Equal(t, "file-helm-legacy-pass", creds.HelmRepoPassword)
}

func TestLoadCredentials_MissingEnvFile(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("GITOPS_USERNAME", "direct-env-user")

	creds, err := LoadCredentials(tempDir)
	require.NoError(t, err)
	assert.Equal(t, "direct-env-user", creds.GitUsername)
	assert.Equal(t, "", creds.GitPassword)
}

func TestLoadCredentials_PlaceholdersIgnored(t *testing.T) {
	tempDir := t.TempDir()

	envContent := `
GITOPS_USERNAME=<...>
GITOPS_PASSWORD=valid-pass
`
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".env"), []byte(envContent), 0600))

	creds, err := LoadCredentials(tempDir)
	require.NoError(t, err)
	assert.Equal(t, "", creds.GitUsername)
	assert.Equal(t, "valid-pass", creds.GitPassword)
}

func TestLoadCredentials_KeyPathResolution(t *testing.T) {
	tempDir := t.TempDir()

	keyPath := filepath.Join(tempDir, "id_rsa")
	require.NoError(t, os.WriteFile(keyPath, []byte("fake-ssh-private-key\n"), 0600))

	envContent := "GITOPS_SSH_PRIVATE_KEY_PATH=id_rsa\n"
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".env"), []byte(envContent), 0600))

	creds, err := LoadCredentials(tempDir)
	require.NoError(t, err)
	assert.Equal(t, "fake-ssh-private-key", creds.GitSSHPrivateKey)
}

func TestLoadCredentials_KeyPathAndInlineConflict(t *testing.T) {
	tempDir := t.TempDir()

	keyPath := filepath.Join(tempDir, "id_rsa")
	require.NoError(t, os.WriteFile(keyPath, []byte("fake-key"), 0600))

	envContent := "GITOPS_SSH_PRIVATE_KEY=inline-key\nGITOPS_SSH_PRIVATE_KEY_PATH=id_rsa\n"
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".env"), []byte(envContent), 0600))

	_, err := LoadCredentials(tempDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot configure both")
}

func TestCredentials_ValidateForCluster(t *testing.T) {
	tests := []struct {
		name    string
		cluster *config.Cluster
		creds   *Credentials
		wantErr string
	}{
		{
			name: "public https repository",
			cluster: &config.Cluster{
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
							Configs: config.Repository{
								URL: "https://github.com/org/repo.git",
							},
						},
					},
				},
			},
			creds:   &Credentials{},
			wantErr: "",
		},
		{
			name: "private https repository with valid credentials",
			cluster: &config.Cluster{
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
							Configs: config.Repository{
								URL: "https://github.com/org/repo.git",
							},
						},
					},
				},
			},
			creds: &Credentials{
				GitUsername: "bot",
				GitPassword: "secret-token",
			},
			wantErr: "",
		},
		{
			name: "private https repository missing password",
			cluster: &config.Cluster{
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
							Configs: config.Repository{
								URL: "https://github.com/org/repo.git",
							},
						},
					},
				},
			},
			creds: &Credentials{
				GitUsername: "bot",
			},
			wantErr: "must either both be set",
		},
		{
			name: "ssh repository missing key",
			cluster: &config.Cluster{
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
							Configs: config.Repository{
								URL: "ssh://git.example.com/org/repo.git",
							},
						},
					},
				},
			},
			creds:   &Credentials{},
			wantErr: "requires GITOPS_SSH_PRIVATE_KEY",
		},
		{
			name: "ssh repository with key",
			cluster: &config.Cluster{
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
							Configs: config.Repository{
								URL: "ssh://git.example.com/org/repo.git",
							},
						},
					},
				},
			},
			creds: &Credentials{
				GitSSHPrivateKey: "ssh-rsa AAAAB3NzaC1yc2E...",
			},
			wantErr: "",
		},
		{
			name: "github app repository missing app private key",
			cluster: &config.Cluster{
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
							Configs: config.Repository{
								URL: "https://github.com/org/repo.git",
							},
						},
					},
				},
			},
			creds: &Credentials{
				GitGitHubAppID:             "12345",
				GitGitHubAppInstallationID: "67890",
			},
			wantErr: "GitHub App authentication requires",
		},
		{
			name: "invalid dockerconfig base64",
			cluster: &config.Cluster{
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
							Configs: config.Repository{
								URL: "https://github.com/org/repo.git",
							},
						},
					},
				},
			},
			creds: &Credentials{
				DockerconfigBase64: "not-valid-base64!@#$",
			},
			wantErr: "invalid DOCKERCONFIG_BASE64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.creds.ValidateForCluster(tt.cluster)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
