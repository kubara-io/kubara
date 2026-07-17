package catalog

import (
	"path/filepath"
	"testing"

	internaltestutil "github.com/kubara-io/kubara/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createCatalogFixturePaths(t *testing.T) (string, string) {
	t.Helper()

	bootstrapPath, generalPath, err := internaltestutil.CreateCatalogFixtures(filepath.Join(t.TempDir(), "catalogs"))
	require.NoError(t, err)

	return bootstrapPath, generalPath
}

func TestLoad_UsesBootstrapCatalogByDefault(t *testing.T) {
	bootstrapPath, _ := createCatalogFixturePaths(t)

	cat, err := Load(LoadOptions{BootstrapCatalog: bootstrapPath})
	require.NoError(t, err)

	assert.Contains(t, cat.Services, "argocd")
	assert.Contains(t, cat.Services, "crds")
}

func TestLoad_MergesGeneralCatalogAfterBootstrap(t *testing.T) {
	bootstrapPath, generalPath := createCatalogFixturePaths(t)

	cat, err := Load(LoadOptions{
		BootstrapCatalog: bootstrapPath,
		Catalogs:         []string{generalPath},
	})
	require.NoError(t, err)

	assert.Contains(t, cat.Services, "argocd")
	assert.Contains(t, cat.Services, "crds")
	assert.Contains(t, cat.Services, "cert-manager")
}
