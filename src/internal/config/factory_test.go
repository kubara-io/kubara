package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultClusterWithCatalog(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("creates normal cluster scaffolding with default placeholders", func(t *testing.T) {
		got, err := NewDefaultClusterWithCatalog(tempDir, testCatalogLoadOptions())
		require.NoError(t, err)

		assert.Equal(t, "<cluster-name>", got.Name)
		assert.Equal(t, "<stage>", got.Stage)
		assert.Equal(t, Hub, got.Type)
		assert.Equal(t, "<subdomain.my-domain.com>", got.DNSName)
		assert.Equal(t, "https://git.example.com/platform/repo.git", got.ArgoCD.Repo.Git.Configs.URL)
		assert.Equal(t, GitAuthModeHTTPS, got.ArgoCD.Repo.AuthMode)
		assert.Equal(t, ClusterSecretsProviderAutoRotator, got.ArgoCD.ClusterSecrets.Provider)
		assert.Equal(t, testClusterCatalogs(), got.Catalogs)
		assert.Equal(t, service.StatusEnabled, got.Services["cert-manager"].Status)
	})
}

func TestInferGitRemote(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("returns empty and https when not in git repository", func(t *testing.T) {
		url, authMode := InferGitRemote(tempDir)
		assert.Empty(t, url)
		assert.Equal(t, GitAuthModeHTTPS, authMode)
	})

	t.Run("infers HTTPS remote and auth mode from git config", func(t *testing.T) {
		repoDir := filepath.Join(tempDir, "https-repo")
		require.NoError(t, os.MkdirAll(repoDir, 0o755))

		cmd := exec.Command("git", "init", repoDir)
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "-C", repoDir, "remote", "add", "origin", "https://github.com/org/repo.git")
		require.NoError(t, cmd.Run())

		url, authMode := InferGitRemote(repoDir)
		assert.Equal(t, "https://github.com/org/repo.git", url)
		assert.Equal(t, GitAuthModeHTTPS, authMode)
	})

	t.Run("infers SSH remote and auth mode from git config", func(t *testing.T) {
		repoDir := filepath.Join(tempDir, "ssh-repo")
		require.NoError(t, os.MkdirAll(repoDir, 0o755))

		cmd := exec.Command("git", "init", repoDir)
		require.NoError(t, cmd.Run())

		cmd = exec.Command("git", "-C", repoDir, "remote", "add", "origin", "ssh://my-git.local/org/repo.git")
		require.NoError(t, cmd.Run())

		url, authMode := InferGitRemote(repoDir)
		assert.Equal(t, "ssh://my-git.local/org/repo.git", url)
		assert.Equal(t, GitAuthModeSSH, authMode)
	})
}

func TestNewDefaultClusterWithCatalog_ReturnsErrorWhenCatalogLoadFails(t *testing.T) {
	_, err := NewDefaultClusterWithCatalog(t.TempDir(), catalog.LoadOptions{
		Catalogs: []string{filepath.Join(t.TempDir(), "does-not-exist")},
	})
	require.Error(t, err)
}

func TestClusterCatalogLoadOptions_DefaultsToGeneralCatalog(t *testing.T) {
	got := clusterCatalogLoadOptions(catalog.LoadOptions{})

	assert.Equal(t, []string{catalog.DefaultGeneralCatalog}, got.Catalogs)
}

func TestClusterCatalogLoadOptions_PreservesExplicitCatalogs(t *testing.T) {
	loadOptions := catalog.LoadOptions{
		BootstrapCatalog: testBootstrapCatalogPath,
		Catalogs:         testClusterCatalogs(),
		Overwrite:        true,
	}

	got := clusterCatalogLoadOptions(loadOptions)

	assert.Equal(t, loadOptions.BootstrapCatalog, got.BootstrapCatalog)
	assert.Equal(t, loadOptions.Catalogs, got.Catalogs)
	assert.Equal(t, loadOptions.Overwrite, got.Overwrite)
	assert.NotSame(t, &loadOptions.Catalogs[0], &got.Catalogs[0])
}

func TestCreateSpokeScaffolding_DefaultsToGeneralCatalog(t *testing.T) {
	cluster := CreateSpokeScaffolding("spoke-a", catalog.LoadOptions{})

	assert.Equal(t, []string{catalog.DefaultGeneralCatalog}, cluster.Catalogs)
	assert.Nil(t, cluster.ArgoCD)
	assert.Equal(t, ClusterNetworking{
		Type: NetworkingIngress,
		Ingress: &IngressNetworking{
			ClassName: "traefik",
		},
	}, cluster.Networking)
}

func TestCreateSpokeScaffolding_PersistsExplicitCatalogs(t *testing.T) {
	cluster := CreateSpokeScaffolding("spoke-a", testCatalogLoadOptions())

	assert.Equal(t, testClusterCatalogs(), cluster.Catalogs)
	assert.Nil(t, cluster.ArgoCD)
}
