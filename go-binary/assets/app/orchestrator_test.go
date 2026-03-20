package app

import (
	"testing"

	"kubara/assets/config"
	"kubara/assets/envmap"

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
							Customer: config.Repository{
								URL:            "https://github.com/old/repo.git",
								TargetRevision: "main",
							},
							Managed: config.Repository{
								URL:            "https://github.com/old/repo.git",
								TargetRevision: "main",
							},
						},
					},
				},
			},
		},
	}
	e := &envmap.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		DomainName:        "example.com",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
		ArgocdHelmRepoUrl: "https://charts.example.com",
	}

	CreateOrUpdateClusterFromEnv(cfg, e)

	require.Len(t, cfg.Clusters, 1)
	updated := cfg.Clusters[0]
	assert.Equal(t, "dev", updated.Stage)
	assert.Equal(t, "kubara-test-dev.example.com", updated.DNSName)
	assert.Equal(t, "kubara-test-dev.example.com", updated.Terraform.DNS.Name)
	assert.Equal(t, "https://github.com/new/repo.git", updated.ArgoCD.Repo.HTTPS.Managed.URL)
	assert.Equal(t, "https://github.com/new/repo.git", updated.ArgoCD.Repo.HTTPS.Customer.URL)
	require.NotNil(t, updated.ArgoCD.HelmRepo)
	assert.Equal(t, "https://charts.example.com", updated.ArgoCD.HelmRepo.URL)
}

func TestCreateOrUpdateClusterFromEnv_CreatesNewClusterWithHelmRepo(t *testing.T) {
	cfg := &config.Config{}
	e := &envmap.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		DomainName:        "example.com",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
		ArgocdHelmRepoUrl: "https://charts.example.com",
	}

	CreateOrUpdateClusterFromEnv(cfg, e)

	require.Len(t, cfg.Clusters, 1)
	cluster := cfg.Clusters[0]
	assert.Equal(t, "https://github.com/new/repo.git", cluster.ArgoCD.Repo.HTTPS.Managed.URL)
	assert.Equal(t, "https://github.com/new/repo.git", cluster.ArgoCD.Repo.HTTPS.Customer.URL)
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
							Customer: config.Repository{
								URL:            "https://github.com/old/repo.git",
								TargetRevision: "main",
							},
							Managed: config.Repository{
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
	e := &envmap.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		DomainName:        "example.com",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
	}

	CreateOrUpdateClusterFromEnv(cfg, e)

	require.Len(t, cfg.Clusters, 1)
	updated := cfg.Clusters[0]
	require.NotNil(t, updated.ArgoCD.HelmRepo)
	assert.Equal(t, "https://charts.old.example.com", updated.ArgoCD.HelmRepo.URL)
}

func TestCreateOrUpdateClusterFromEnv_CreatesNewClusterWithoutHelmRepoWhenEnvMissing(t *testing.T) {
	cfg := &config.Config{}
	e := &envmap.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		DomainName:        "example.com",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
	}

	CreateOrUpdateClusterFromEnv(cfg, e)

	require.Len(t, cfg.Clusters, 1)
	cluster := cfg.Clusters[0]
	assert.Nil(t, cluster.ArgoCD.HelmRepo)
}

func TestCreateOrUpdateClusterFromEnv_NormalizesOCIHelmRepoURL(t *testing.T) {
	cfg := &config.Config{}
	e := &envmap.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		DomainName:        "example.com",
		ArgocdGitHttpsUrl: "https://github.com/new/repo.git",
		ArgocdHelmRepoUrl: "oci://registry-1.docker.io/bitnamicharts",
	}

	CreateOrUpdateClusterFromEnv(cfg, e)

	require.Len(t, cfg.Clusters, 1)
	cluster := cfg.Clusters[0]
	require.NotNil(t, cluster.ArgoCD.HelmRepo)
	assert.Equal(t, "registry-1.docker.io/bitnamicharts", cluster.ArgoCD.HelmRepo.URL)
}
