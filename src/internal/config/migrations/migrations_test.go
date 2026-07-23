package migrations

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateV1Alpha2FilesCleansUpEmptyLegacyCategoryDirs(t *testing.T) {
	tempDir := t.TempDir()

	helmSource := filepath.Join(tempDir, "customer-service-catalog", "helm", "test-cluster", "argo-cd")
	terraformSource := filepath.Join(tempDir, "customer-service-catalog", "terraform", "test-cluster")
	otherTerraformSource := filepath.Join(tempDir, "customer-service-catalog", "terraform", "other-cluster")

	require.NoError(t, os.MkdirAll(helmSource, 0o755))
	require.NoError(t, os.MkdirAll(terraformSource, 0o755))
	require.NoError(t, os.MkdirAll(otherTerraformSource, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(helmSource, "values.yaml"), []byte("kind: values"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(helmSource, "values.generated.yaml"), []byte("generated"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(helmSource, "additional-values.yaml"), []byte("additional"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(terraformSource, "main.tf"), []byte("resource {}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(otherTerraformSource, "keep.tf"), []byte("keep"), 0o644))

	require.NoError(t, migrateV1Alpha2Files(tempDir, "test-cluster"))

	assert.NoFileExists(t, filepath.Join(tempDir, "customer-service-catalog", "test-cluster", "helm", "argo-cd", "values.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "customer-service-catalog", "test-cluster", "helm", "argo-cd", "values.generated.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "customer-service-catalog", "test-cluster", "helm", "argo-cd", "values-additional.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "customer-service-catalog", "test-cluster", "terraform", "main.tf"))
	assert.NoDirExists(t, filepath.Join(tempDir, "customer-service-catalog", "helm"))
	assert.NoDirExists(t, filepath.Join(tempDir, "customer-service-catalog", "terraform", "test-cluster"))
	assert.DirExists(t, filepath.Join(tempDir, "customer-service-catalog", "terraform"))
	assert.FileExists(t, filepath.Join(otherTerraformSource, "keep.tf"))
}

func TestMigrateV1Alpha4ConfigRenamesRepoAndDNS(t *testing.T) {
	config := map[string]any{
		"version": ConfigVersionV1Alpha4,
		"clusters": []any{
			map[string]any{
				"name":    "test-cluster",
				"dnsName": "example.com",
				"argocd": map[string]any{
					"repo": map[string]any{
						"oci": map[string]any{
							"configs": map[string]any{"url": "ghcr.io/example/configs"},
						},
						"https": map[string]any{
							"configs":    map[string]any{"url": "https://github.com/example/configs.git"},
							"components": map[string]any{"url": "https://github.com/example/components.git"},
						},
					},
				},
				"terraform": map[string]any{
					"projectId": "some-id",
					"dns": map[string]any{
						"name":  "example.com",
						"email": "admin@example.com",
					},
				},
			},
		},
	}

	require.NoError(t, migrateV1Alpha4Config(config))

	assert.Equal(t, ConfigVersionV1Alpha5, config["version"])

	cluster := config["clusters"].([]any)[0].(map[string]any)
	repo := cluster["argocd"].(map[string]any)["repo"].(map[string]any)
	assert.Contains(t, repo, "git")
	assert.NotContains(t, repo, "https")
	assert.Contains(t, repo["git"].(map[string]any), "configs")
	assert.Contains(t, repo, "oci")

	terraform := cluster["terraform"].(map[string]any)
	assert.NotContains(t, terraform, "dns")
	assert.Equal(t, "admin@example.com", terraform["dnsContactEmail"])
}

func TestMigrateV1Alpha4ConfigWarnsWhenLegacyDNSNameDiffers(t *testing.T) {
	var logOutput bytes.Buffer
	previousLogger := log.Logger
	log.Logger = zerolog.New(&logOutput)
	t.Cleanup(func() {
		log.Logger = previousLogger
	})

	config := map[string]any{
		"version": ConfigVersionV1Alpha4,
		"clusters": []any{
			map[string]any{
				"name":    "test-cluster",
				"dnsName": "new.example.com",
				"terraform": map[string]any{
					"dns": map[string]any{
						"name": "legacy.example.com",
					},
				},
			},
		},
	}

	require.NoError(t, migrateV1Alpha4Config(config))
	assert.Contains(t, logOutput.String(), "terraform.dns.name differs from the cluster dnsName")
	assert.Contains(t, logOutput.String(), `"legacyDnsName":"legacy.example.com"`)
	assert.Contains(t, logOutput.String(), `"dnsName":"new.example.com"`)
}

func TestMigrateV1Alpha4ConfigRejectsConflictingKeys(t *testing.T) {
	config := map[string]any{
		"version": ConfigVersionV1Alpha4,
		"clusters": []any{
			map[string]any{
				"name": "test-cluster",
				"argocd": map[string]any{
					"repo": map[string]any{
						"https": map[string]any{"configs": map[string]any{}},
						"git":   map[string]any{"configs": map[string]any{}},
					},
				},
			},
		},
	}

	err := migrateV1Alpha4Config(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both legacy https and git")
}

func TestMigrateV1Alpha2ConfigMigratesReposAndCatalogDirs(t *testing.T) {
	tempDir := t.TempDir()

	customerHelmSource := filepath.Join(tempDir, "customer-service-catalog", "helm", "test-cluster", "argo-cd")
	managedTerraformSource := filepath.Join(tempDir, "managed-service-catalog", "terraform", "stackit")
	require.NoError(t, os.MkdirAll(customerHelmSource, 0o755))
	require.NoError(t, os.MkdirAll(managedTerraformSource, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(customerHelmSource, "values.yaml"), []byte("kind: values"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(customerHelmSource, "values.generated.yaml"), []byte("generated"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(managedTerraformSource, "main.tf"), []byte("resource {}"), 0o644))

	config := map[string]any{
		"version": ConfigVersionV1Alpha2,
		"clusters": []any{
			map[string]any{
				"name": "test-cluster",
				"argocd": map[string]any{
					"repo": map[string]any{
						"https": map[string]any{
							"customer": map[string]any{"url": "https://github.com/example/configs.git", "path": "customer-service-catalog/helm"},
							"managed":  map[string]any{"url": "https://github.com/example/components.git"},
						},
						"oci": map[string]any{
							"customer": map[string]any{"url": "ghcr.io/example/configs", "path": "platform-configs/helm"},
							"managed":  map[string]any{"url": "ghcr.io/example/components"},
						},
					},
				},
			},
		},
	}

	require.NoError(t, migrateV1Alpha2Config(tempDir, config))

	assert.Equal(t, ConfigVersionV1Alpha3, config["version"])

	cluster := config["clusters"].([]any)[0].(map[string]any)
	repo := cluster["argocd"].(map[string]any)["repo"].(map[string]any)
	for _, protocol := range []string{"https", "oci"} {
		repoConfig := repo[protocol].(map[string]any)
		assert.Contains(t, repoConfig, "configs")
		assert.Contains(t, repoConfig, "components")
		assert.NotContains(t, repoConfig, "customer")
		assert.NotContains(t, repoConfig, "managed")
		assert.Equal(t, "platform-configs", repoConfig["configs"].(map[string]any)["path"])
	}

	assert.NoFileExists(t, filepath.Join(tempDir, "platform-configs", "test-cluster", "helm", "argo-cd", "values.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "platform-configs", "test-cluster", "helm", "argo-cd", "values.generated.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "platform-components", "terraform", "stackit", "main.tf"))
	assert.NoDirExists(t, filepath.Join(tempDir, "customer-service-catalog"))
	assert.NoDirExists(t, filepath.Join(tempDir, "managed-service-catalog"))
}

func TestMigrateLegacyValuesFilesScopesToKnownCatalogRoots(t *testing.T) {
	tempDir := t.TempDir()

	customerLegacyDir := filepath.Join(tempDir, "customer-service-catalog", "helm", "test-cluster", "argo-cd")
	customerCurrentDir := filepath.Join(tempDir, "platform-configs", "helm", "test-cluster", "argo-cd")
	managedLegacyDir := filepath.Join(tempDir, "managed-service-catalog", "helm", "argo-cd")
	managedCurrentDir := filepath.Join(tempDir, "platform-components", "helm", "argo-cd")
	unrelatedDir := filepath.Join(tempDir, "unrelated-project", "argo-cd")

	require.NoError(t, os.MkdirAll(customerLegacyDir, 0o755))
	require.NoError(t, os.MkdirAll(customerCurrentDir, 0o755))
	require.NoError(t, os.MkdirAll(managedLegacyDir, 0o755))
	require.NoError(t, os.MkdirAll(managedCurrentDir, 0o755))
	require.NoError(t, os.MkdirAll(unrelatedDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(customerLegacyDir, "values.yaml"), []byte("legacy customer"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(customerCurrentDir, "values.yaml"), []byte("current customer"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(managedLegacyDir, "values.yaml"), []byte("legacy managed"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(managedCurrentDir, "values.yaml"), []byte("current managed"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(unrelatedDir, "values.yaml"), []byte("outside scope"), 0o644))

	require.NoError(t, migrateLegacyValuesFiles(tempDir))

	assert.NoFileExists(t, filepath.Join(customerLegacyDir, "values.yaml"))
	assert.FileExists(t, filepath.Join(customerLegacyDir, "values.generated.yaml"))
	assert.FileExists(t, filepath.Join(customerCurrentDir, "values.yaml"))
	assert.NoFileExists(t, filepath.Join(customerCurrentDir, "values.generated.yaml"))
	assert.FileExists(t, filepath.Join(managedLegacyDir, "values.yaml"))
	assert.NoFileExists(t, filepath.Join(managedLegacyDir, "values.generated.yaml"))
	assert.FileExists(t, filepath.Join(managedCurrentDir, "values.yaml"))
	assert.NoFileExists(t, filepath.Join(managedCurrentDir, "values.generated.yaml"))
	assert.FileExists(t, filepath.Join(unrelatedDir, "values.yaml"))
	assert.NoFileExists(t, filepath.Join(unrelatedDir, "values.generated.yaml"))
}

func TestMigrateV1Alpha2ClusterRejectsNonObjectRepos(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		wantErr  string
	}{
		{
			name:     "https repo must be an object",
			protocol: "https",
			wantErr:  "cannot migrate HTTPS repo: repo must be an object",
		},
		{
			name:     "oci repo must be an object",
			protocol: "oci",
			wantErr:  "cannot migrate OCI repo: repo must be an object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := map[string]any{
				"name": "test-cluster",
				"argocd": map[string]any{
					"repo": map[string]any{
						tt.protocol: "not-an-object",
					},
				},
			}

			err := migrateV1Alpha2Cluster(cluster, 0)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestMigrateV1Alpha2ConfigRejectsInvalidClusterName(t *testing.T) {
	config := map[string]any{
		"version": ConfigVersionV1Alpha2,
		"clusters": []any{
			map[string]any{
				"name": 123,
			},
		},
	}

	err := migrateV1Alpha2Config(t.TempDir(), config)
	require.Error(t, err)
	assert.ErrorContains(t, err, `clusters[0].name must be a non-empty string`)
}

func TestMigrateV1Alpha2RepoKeepsCurrentKeys(t *testing.T) {
	repo := map[string]any{
		"configs":    map[string]any{"url": "https://github.com/example/configs.git"},
		"components": map[string]any{"url": "https://github.com/example/components.git"},
	}

	require.NoError(t, migrateV1Alpha2Repo(repo))

	assert.Equal(t, map[string]any{"url": "https://github.com/example/configs.git"}, repo["configs"])
	assert.Equal(t, map[string]any{"url": "https://github.com/example/components.git"}, repo["components"])
	assert.NotContains(t, repo, "customer")
	assert.NotContains(t, repo, "managed")
}

func TestMigrateV1Alpha3Config(t *testing.T) {
	config := map[string]any{
		"version": ConfigVersionV1Alpha3,
		"clusters": []any{
			map[string]any{
				"name": "test-cluster",
				"services": map[string]any{
					"argocd": map[string]any{
						"status": "disabled",
					},
				},
			},
		},
	}

	require.NoError(t, migrateV1Alpha3Config(config))
	assert.Equal(t, ConfigVersionV1Alpha4, config["version"])

	cluster := config["clusters"].([]any)[0].(map[string]any)
	assert.Equal(t, []any{"oci://ghcr.io/kubara-io/catalogs/general:1.1.0"}, cluster["catalogs"])
	assert.Equal(t, "disabled", cluster["argocd"].(map[string]any)["selfManaged"])
	assert.NotContains(t, cluster["services"], "argocd")
}
