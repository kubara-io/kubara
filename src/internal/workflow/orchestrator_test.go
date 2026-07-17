package workflow

import (
	"path/filepath"
	"testing"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateOrUpdateClusterFromEnv_UpdatesExistingClusterIncludingHelmRepo(t *testing.T) {
	cfg := &config.Config{
		Clusters: []config.Cluster{
			{
				Name:    "kubara-test",
				Stage:   "stage",
				DNSName: "kubara-test-stage.example.com",
				Terraform: &config.Terraform{
					DNS: config.DNS{
						Name: "kubara-test-stage.example.com",
					},
				},
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						HTTPS: &config.RepoType{
							Configs: config.Repository{
								URL:            "https://github.com/old/repo.git",
								TargetRevision: "main",
							},
							Components: config.Repository{
								URL:            "https://github.com/old/repo.git",
								TargetRevision: "main",
							},
						},
					},
				},
			},
		},
	}
	e := &envconfig.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
		ArgocdHelmRepoUrl: "https://charts.example.com",
	}

	err := CreateOrUpdateClusterFromEnvWithCatalog(cfg, e, catalog.LoadOptions{})
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	updated := cfg.Clusters[0]
	assert.Equal(t, "dev", updated.Stage)
	assert.Equal(t, "kubara-test-stage.example.com", updated.DNSName)
	assert.Equal(t, "kubara-test-stage.example.com", updated.Terraform.DNS.Name)
	assert.Equal(t, "https://github.com/new/repo.git", updated.ArgoCD.Repo.HTTPS.Components.URL)
	assert.Equal(t, "https://github.com/new/repo.git", updated.ArgoCD.Repo.HTTPS.Configs.URL)
	require.NotNil(t, updated.ArgoCD.HelmRepo)
	assert.Equal(t, "https://charts.example.com", updated.ArgoCD.HelmRepo.URL)
}

func TestCreateOrUpdateClusterFromEnv_UpdatesExistingClusterWithoutTerraform(t *testing.T) {
	cfg := &config.Config{
		Clusters: []config.Cluster{
			{
				Name:      "kubara-test",
				Stage:     "stage",
				DNSName:   "kubara-test-stage.example.com",
				Terraform: nil,
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						HTTPS: &config.RepoType{
							Configs: config.Repository{
								URL:            "https://github.com/old/repo.git",
								TargetRevision: "main",
							},
							Components: config.Repository{
								URL:            "https://github.com/old/repo.git",
								TargetRevision: "main",
							},
						},
					},
				},
			},
		},
	}
	e := &envconfig.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
	}

	err := CreateOrUpdateClusterFromEnvWithCatalog(cfg, e, catalog.LoadOptions{})
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	updated := cfg.Clusters[0]
	assert.Equal(t, "dev", updated.Stage)
	assert.Equal(t, "kubara-test-stage.example.com", updated.DNSName)
	assert.Nil(t, updated.Terraform)
	assert.Equal(t, "https://github.com/new/repo.git", updated.ArgoCD.Repo.HTTPS.Components.URL)
	assert.Equal(t, "https://github.com/new/repo.git", updated.ArgoCD.Repo.HTTPS.Configs.URL)
}

func TestCreateOrUpdateClusterFromEnv_CreatesNewClusterWithHelmRepo(t *testing.T) {
	cfg := &config.Config{}
	e := &envconfig.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
		ArgocdHelmRepoUrl: "https://charts.example.com",
	}

	err := CreateOrUpdateClusterFromEnvWithCatalog(cfg, e, catalog.LoadOptions{})
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	cluster := cfg.Clusters[0]
	assert.Equal(t, "https://github.com/new/repo.git", cluster.ArgoCD.Repo.HTTPS.Components.URL)
	assert.Equal(t, "https://github.com/new/repo.git", cluster.ArgoCD.Repo.HTTPS.Configs.URL)
	require.NotNil(t, cluster.ArgoCD.HelmRepo)
	assert.Equal(t, "https://charts.example.com", cluster.ArgoCD.HelmRepo.URL)
}

func TestCreateOrUpdateClusterFromEnv_DoesNotOverrideHelmRepoWhenEnvMissing(t *testing.T) {
	cfg := &config.Config{
		Clusters: []config.Cluster{
			{
				Name:    "kubara-test",
				Stage:   "stage",
				DNSName: "kubara-test-stage.example.com",
				Terraform: &config.Terraform{
					DNS: config.DNS{
						Name: "kubara-test-stage.example.com",
					},
				},
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						HTTPS: &config.RepoType{
							Configs: config.Repository{
								URL:            "https://github.com/old/repo.git",
								TargetRevision: "main",
							},
							Components: config.Repository{
								URL:            "https://github.com/old/repo.git",
								TargetRevision: "main",
							},
						},
					},
					HelmRepo: &config.HelmRepository{
						URL: "https://charts.old.example.com",
					},
				},
			},
		},
	}
	e := &envconfig.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
	}

	err := CreateOrUpdateClusterFromEnvWithCatalog(cfg, e, catalog.LoadOptions{})
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	updated := cfg.Clusters[0]
	require.NotNil(t, updated.ArgoCD.HelmRepo)
	assert.Equal(t, "https://charts.old.example.com", updated.ArgoCD.HelmRepo.URL)
}

func TestCreateOrUpdateClusterFromEnv_CreatesNewClusterWithoutHelmRepoWhenEnvMissing(t *testing.T) {
	cfg := &config.Config{}
	e := &envconfig.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
	}

	err := CreateOrUpdateClusterFromEnvWithCatalog(cfg, e, catalog.LoadOptions{})
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	cluster := cfg.Clusters[0]
	assert.Nil(t, cluster.ArgoCD.HelmRepo)
}

func TestCreateOrUpdateClusterFromEnv_NormalizesOCIHelmRepoURL(t *testing.T) {
	cfg := &config.Config{}
	e := &envconfig.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
		ArgocdHelmRepoUrl: "oci://registry-1.docker.io/bitnamicharts",
	}

	err := CreateOrUpdateClusterFromEnvWithCatalog(cfg, e, catalog.LoadOptions{})
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	cluster := cfg.Clusters[0]
	require.NotNil(t, cluster.ArgoCD.HelmRepo)
	assert.Equal(t, "registry-1.docker.io/bitnamicharts", cluster.ArgoCD.HelmRepo.URL)
}

func TestCreateOrUpdateClusterFromEnvWithCatalog_ReturnsErrorWhenCatalogLoadFails(t *testing.T) {
	cfg := &config.Config{}
	e := &envconfig.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
	}

	err := CreateOrUpdateClusterFromEnvWithCatalog(cfg, e, catalog.LoadOptions{
		Catalogs: []string{filepath.Join(t.TempDir(), "does-not-exist")},
	})
	require.Error(t, err)
	require.Empty(t, cfg.Clusters)
}
