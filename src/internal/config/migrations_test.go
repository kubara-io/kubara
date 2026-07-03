package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateV1Alpha2FilesCleansUpEmptyLegacyCategoryDirs(t *testing.T) {
	tempDir := t.TempDir()

	helmSource := filepath.Join(tempDir, "customer-service-catalog", "helm", "test-cluster", "argo-cd")
	customHelmSource := filepath.Join(tempDir, "customer-service-catalog", "helm", "test-cluster", "custom-app")
	terraformSource := filepath.Join(tempDir, "customer-service-catalog", "terraform", "test-cluster")
	otherTerraformSource := filepath.Join(tempDir, "customer-service-catalog", "terraform", "other-cluster")

	require.NoError(t, os.MkdirAll(helmSource, 0o755))
	require.NoError(t, os.MkdirAll(customHelmSource, 0o755))
	require.NoError(t, os.MkdirAll(terraformSource, 0o755))
	require.NoError(t, os.MkdirAll(otherTerraformSource, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(helmSource, "values.yaml"), []byte("kind: values"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(helmSource, "values.generated.yaml"), []byte("generated"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(helmSource, "additional-values.yaml"), []byte("additional"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(customHelmSource, "values.yaml"), []byte("custom: true"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(customHelmSource, "additional-values.yaml"), []byte("custom: additional"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(terraformSource, "main.tf"), []byte("resource {}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(otherTerraformSource, "keep.tf"), []byte("keep"), 0o644))

	require.NoError(t, migrateV1Alpha2Files(tempDir, "test-cluster"))

	assert.NoFileExists(t, filepath.Join(tempDir, "customer-service-catalog", "test-cluster", "helm", "argo-cd", "values.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "customer-service-catalog", "test-cluster", "helm", "argo-cd", "values.generated.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "customer-service-catalog", "test-cluster", "helm", "argo-cd", "values-additional.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "customer-service-catalog", "test-cluster", "helm", "custom-app", "values.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "customer-service-catalog", "test-cluster", "helm", "custom-app", "additional-values.yaml"))
	assert.NoFileExists(t, filepath.Join(tempDir, "customer-service-catalog", "test-cluster", "helm", "custom-app", "values-additional.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "customer-service-catalog", "test-cluster", "terraform", "main.tf"))
	assert.NoDirExists(t, filepath.Join(tempDir, "customer-service-catalog", "helm"))
	assert.NoDirExists(t, filepath.Join(tempDir, "customer-service-catalog", "terraform", "test-cluster"))
	assert.DirExists(t, filepath.Join(tempDir, "customer-service-catalog", "terraform"))
	assert.FileExists(t, filepath.Join(otherTerraformSource, "keep.tf"))
}

func TestMigrateV1Alpha2ConfigMigratesReposAndCatalogDirs(t *testing.T) {
	tempDir := t.TempDir()

	customerHelmSource := filepath.Join(tempDir, "customer-service-catalog", "helm", "test-cluster", "argo-cd")
	managedTerraformSource := filepath.Join(tempDir, "managed-service-catalog", "terraform", "test-cluster")
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
							"customer": map[string]any{"url": "https://github.com/example/configs.git"},
							"managed":  map[string]any{"url": "https://github.com/example/components.git"},
						},
						"oci": map[string]any{
							"customer": map[string]any{"url": "ghcr.io/example/configs"},
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
	}

	assert.NoFileExists(t, filepath.Join(tempDir, "platform-configs", "test-cluster", "helm", "argo-cd", "values.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "platform-configs", "test-cluster", "helm", "argo-cd", "values.generated.yaml"))
	assert.FileExists(t, filepath.Join(tempDir, "platform-components", "test-cluster", "terraform", "main.tf"))
	assert.NoDirExists(t, filepath.Join(tempDir, "customer-service-catalog"))
	assert.NoDirExists(t, filepath.Join(tempDir, "managed-service-catalog"))
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
