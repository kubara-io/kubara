package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageCatalog_IncludesCatalogDirectoryContents(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	workDir := t.TempDir()
	catalogRoot := filepath.Join(t.TempDir(), "demo-catalog")
	writeCatalogFixture(t, catalogRoot, "demo-catalog", "1.2.3")
	require.NoError(t, os.MkdirAll(filepath.Join(catalogRoot, "customer-service-catalog", "helm"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(catalogRoot, "customer-service-catalog", "helm", "values.yaml"), []byte("foo: bar\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(catalogRoot, "notes.txt"), []byte("package me"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(catalogRoot, "scratch"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(catalogRoot, "scratch", "debug.yaml"), []byte("debug: true\n"), 0o600))

	result, err := PackageCatalog(PackageOptions{CatalogRoot: catalogRoot})
	require.NoError(t, err)

	unpackResult, err := UnpackageCatalog(UnpackageOptions{
		Reference: result.Reference,
		WorkDir:   workDir,
	})
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(unpackResult.OutputPath, "Catalog.yaml"))
	assert.FileExists(t, filepath.Join(unpackResult.OutputPath, "services", "demo-service.yaml"))
	assert.FileExists(t, filepath.Join(unpackResult.OutputPath, "customer-service-catalog", "helm", "values.yaml"))
	assert.FileExists(t, filepath.Join(unpackResult.OutputPath, "notes.txt"))
	assert.FileExists(t, filepath.Join(unpackResult.OutputPath, "scratch", "debug.yaml"))
}

func writeCatalogFixture(t *testing.T, root, name, version string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Join(root, "services"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "managed-service-catalog", "helm"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "customer-service-catalog", "helm"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "Catalog.yaml"), []byte(`
apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: `+name+`
spec:
  version: `+version+`
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "services", "demo-service.yaml"), []byte(`
apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: demo-service
spec:
  chartPath: demo-service
  status: enabled
`), 0o600))
}
