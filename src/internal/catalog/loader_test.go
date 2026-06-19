package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBuiltIn(t *testing.T) {
	cat, err := LoadBuiltIn()
	require.NoError(t, err)

	assert.NotEmpty(t, cat.Services)
	assert.Contains(t, cat.Services, "cert-manager")
	assert.Contains(t, cat.Services, "argocd")
}

func TestLoad_ExternalCatalogOverwriteBehavior(t *testing.T) {
	tempDir := t.TempDir()
	writeLoaderServiceFixture(t, tempDir, "argocd", "custom-argo-cd")

	_, err := Load(LoadOptions{CatalogPath: tempDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	cat, err := Load(LoadOptions{CatalogPath: tempDir, Overwrite: true})
	require.NoError(t, err)
	assert.Equal(t, "custom-argo-cd", cat.Services["argocd"].Spec.ChartPath)
}

func TestLoad_MissingServicesDirectory(t *testing.T) {
	tempDir := t.TempDir()

	_, err := Load(LoadOptions{CatalogPath: tempDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `catalog services path`)
	assert.Contains(t, err.Error(), `does not exist`)
}

func writeLoaderServiceFixture(t *testing.T, root, name, chartPath string) {
	t.Helper()

	servicesDir := filepath.Join(root, "services")
	require.NoError(t, os.MkdirAll(servicesDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(servicesDir, name+".yaml"), []byte(`
apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: `+name+`
spec:
  chartPath: `+chartPath+`
  status: enabled
`), 0o644))
}
