package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	internaltestutil "github.com/kubara-io/kubara/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCatalogLoadOptions(t *testing.T) catalog.LoadOptions {
	t.Helper()

	root, err := os.MkdirTemp("", "kubara-workflow-catalog-tests-*")
	require.NoError(t, err)

	bootstrapPath, generalPath, err := internaltestutil.CreateCatalogFixtures(filepath.Join(root, "catalogs"))
	require.NoError(t, err)

	return catalog.LoadOptions{
		BootstrapCatalog: bootstrapPath,
		Catalogs:         []string{generalPath},
	}
}

func TestCreateOrUpdateCluster_UpdatesExistingClusterIncludingHelmRepo(t *testing.T) {
	cfg := &config.Config{
		Clusters: []config.Cluster{
			{
				Name:    "kubara-test",
				Stage:   "stage",
				DNSName: "kubara-test-stage.example.com",
				Terraform: &config.Terraform{
					DNSContactEmail: "admin@example.com",
				},
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
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

	err := CreateOrUpdateCluster(cfg, e, testCatalogLoadOptions(t))
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	updated := cfg.Clusters[0]
	assert.Equal(t, "dev", updated.Stage)
	assert.Equal(t, "kubara-test-stage.example.com", updated.DNSName)
	assert.Equal(t, "admin@example.com", updated.Terraform.DNSContactEmail)
	assert.Equal(t, "https://github.com/new/repo.git", updated.ArgoCD.Repo.Git.Components.URL)
	assert.Equal(t, "https://github.com/new/repo.git", updated.ArgoCD.Repo.Git.Configs.URL)
	require.NotNil(t, updated.ArgoCD.HelmRepo)
	assert.Equal(t, "https://charts.example.com", updated.ArgoCD.HelmRepo.URL)
}

func TestCreateOrUpdateCluster_UpdatesExistingClusterWithoutTerraform(t *testing.T) {
	cfg := &config.Config{
		Clusters: []config.Cluster{
			{
				Name:      "kubara-test",
				Stage:     "stage",
				DNSName:   "kubara-test-stage.example.com",
				Terraform: nil,
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
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

	err := CreateOrUpdateCluster(cfg, e, testCatalogLoadOptions(t))
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	updated := cfg.Clusters[0]
	assert.Equal(t, "dev", updated.Stage)
	assert.Equal(t, "kubara-test-stage.example.com", updated.DNSName)
	assert.Nil(t, updated.Terraform)
	assert.Equal(t, "https://github.com/new/repo.git", updated.ArgoCD.Repo.Git.Components.URL)
	assert.Equal(t, "https://github.com/new/repo.git", updated.ArgoCD.Repo.Git.Configs.URL)
}

func TestCreateOrUpdateCluster_CreatesNewClusterWithHelmRepo(t *testing.T) {
	cfg := &config.Config{}
	e := &envconfig.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
		ArgocdHelmRepoUrl: "https://charts.example.com",
	}

	err := CreateOrUpdateCluster(cfg, e, testCatalogLoadOptions(t))
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	cluster := cfg.Clusters[0]
	assert.Equal(t, "https://github.com/new/repo.git", cluster.ArgoCD.Repo.Git.Components.URL)
	assert.Equal(t, "https://github.com/new/repo.git", cluster.ArgoCD.Repo.Git.Configs.URL)
	require.NotNil(t, cluster.ArgoCD.HelmRepo)
	assert.Equal(t, "https://charts.example.com", cluster.ArgoCD.HelmRepo.URL)
}

func TestCreateOrUpdateCluster_DoesNotOverrideHelmRepoWhenEnvMissing(t *testing.T) {
	cfg := &config.Config{
		Clusters: []config.Cluster{
			{
				Name:    "kubara-test",
				Stage:   "stage",
				DNSName: "kubara-test-stage.example.com",
				Terraform: &config.Terraform{
					DNSContactEmail: "admin@example.com",
				},
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
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

	err := CreateOrUpdateCluster(cfg, e, testCatalogLoadOptions(t))
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	updated := cfg.Clusters[0]
	require.NotNil(t, updated.ArgoCD.HelmRepo)
	assert.Equal(t, "https://charts.old.example.com", updated.ArgoCD.HelmRepo.URL)
}

func TestCreateOrUpdateCluster_CreatesNewClusterWithoutHelmRepoWhenEnvMissing(t *testing.T) {
	cfg := &config.Config{}
	e := &envconfig.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
	}

	err := CreateOrUpdateCluster(cfg, e, testCatalogLoadOptions(t))
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	cluster := cfg.Clusters[0]
	assert.Nil(t, cluster.ArgoCD.HelmRepo)
}

func TestCreateOrUpdateCluster_NormalizesOCIHelmRepoURL(t *testing.T) {
	cfg := &config.Config{}
	e := &envconfig.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
		ArgocdHelmRepoUrl: "oci://registry-1.docker.io/bitnamicharts",
	}

	err := CreateOrUpdateCluster(cfg, e, testCatalogLoadOptions(t))
	require.NoError(t, err)

	require.Len(t, cfg.Clusters, 1)
	cluster := cfg.Clusters[0]
	require.NotNil(t, cluster.ArgoCD.HelmRepo)
	assert.Equal(t, "registry-1.docker.io/bitnamicharts", cluster.ArgoCD.HelmRepo.URL)
}

func TestCreateOrUpdateCluster_ReturnsErrorWhenCatalogLoadFails(t *testing.T) {
	cfg := &config.Config{}
	e := &envconfig.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
	}

	loadOptions := testCatalogLoadOptions(t)
	loadOptions.Catalogs = []string{filepath.Join(t.TempDir(), "does-not-exist")}

	err := CreateOrUpdateCluster(cfg, e, loadOptions)
	require.Error(t, err)
	require.Empty(t, cfg.Clusters)
}
