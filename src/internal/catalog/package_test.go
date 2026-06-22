package catalog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/content/oci"
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

func TestPackageCatalog_RequiresCatalogManifest(t *testing.T) {
	catalogRoot := t.TempDir()

	_, err := PackageCatalog(PackageOptions{CatalogRoot: catalogRoot})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCatalogManifestNotFound)
	assert.EqualError(t, err, `catalog root "`+catalogRoot+`" is missing Catalog.yaml: catalog manifest not found`)
}

func TestPackageCatalog_PipesCatalogAnnotationsIntoOCIManifest(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	catalogRoot := filepath.Join(t.TempDir(), "demo-catalog")
	writeCatalogFixture(t, catalogRoot, "demo-catalog", "1.2.3")
	require.NoError(t, os.WriteFile(filepath.Join(catalogRoot, "Catalog.yaml"), []byte(`
apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: demo-catalog
  annotations:
    org.opencontainers.image.source: https://github.com/kubara-io/kubara
    io.kubara.catalog.version: should-not-win
spec:
  version: 1.2.3
`), 0o600))

	result, err := PackageCatalog(PackageOptions{CatalogRoot: catalogRoot})
	require.NoError(t, err)

	annotations := readCachedManifestAnnotations(t, result.Artifact)
	assert.Equal(t, "https://github.com/kubara-io/kubara", annotations["org.opencontainers.image.source"])
	assert.Equal(t, "demo-catalog", annotations["io.kubara.catalog.name"])
	assert.Equal(t, "1.2.3", annotations["io.kubara.catalog.version"])
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

func readCachedManifestAnnotations(t *testing.T, artifact CachedArtifact) map[string]string {
	t.Helper()

	artifactDir, err := artifactDirPath(artifact.ManifestDigest)
	require.NoError(t, err)

	layoutStore, err := oci.New(filepath.Join(artifactDir, "layout"))
	require.NoError(t, err)

	desc, err := layoutStore.Resolve(context.Background(), artifact.ManifestDigest)
	require.NoError(t, err)

	reader, err := layoutStore.Fetch(context.Background(), desc)
	require.NoError(t, err)
	defer func() {
		_ = reader.Close()
	}()

	var manifest v1.Manifest
	require.NoError(t, json.NewDecoder(reader).Decode(&manifest))

	return manifest.Annotations
}
