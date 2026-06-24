package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushCatalogRejectsDestinationVersionMismatch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	catalogRoot := writeTestCatalogManifest(t, "example", "1.2.3")
	packaged, err := PackageCatalog(PackageOptions{
		CatalogRoot: catalogRoot,
	})
	require.NoError(t, err)

	_, err = PushCatalog(context.Background(), PushOptions{
		Reference: "oci://registry.example.com/catalogs/example:9.9.9",
		From:      packaged.Reference,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `destination tag "9.9.9" must match catalog version "1.2.3"`)
}

func TestEnsureReferenceTagMatchesCatalogVersionRejectsPullVersionMismatch(t *testing.T) {
	ref, err := ParseOCIReference("oci://registry.example.com/catalogs/example:1.2.3")
	require.NoError(t, err)

	err = ensureReferenceTagMatchesCatalogVersion(ref, "9.9.9", "pull")
	require.Error(t, err)
	assert.EqualError(t, err, `pull tag "1.2.3" must match catalog version "9.9.9"`)
}

func TestEnsureReferenceTagMatchesCatalogVersionAllowsDigestReferences(t *testing.T) {
	ref, err := ParseOCIReference("oci://registry.example.com/catalogs/example@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.NoError(t, err)

	err = ensureReferenceTagMatchesCatalogVersion(ref, "9.9.9", "pull")
	require.NoError(t, err)
}

func TestListCachedCatalogsIncludesDigestReferences(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	ref, err := ParseOCIReference("oci://registry.example.com/catalogs/example@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	require.NoError(t, err)

	artifact := CachedArtifact{
		SchemaVersion:  cacheSchemaVersion,
		CatalogName:    "example",
		CatalogVersion: "1.2.3",
		ManifestDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		RootDirectory:  "example",
	}
	require.NoError(t, writeCachedReference(ref, artifact))

	entries, err := ListCachedCatalogs()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, CachedCatalogEntry{
		Reference:      ref.Raw,
		CatalogName:    artifact.CatalogName,
		CatalogVersion: artifact.CatalogVersion,
		ManifestDigest: artifact.ManifestDigest,
	}, entries[0])
}

func TestPushCatalogRequiresCachedDestinationReferenceWhenFromIsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := PushCatalog(context.Background(), PushOptions{
		Reference: "oci://registry.example.com/catalogs/example:1.2.3",
	})
	require.Error(t, err)
	assert.EqualError(t, err, `cached catalog "oci://registry.example.com/catalogs/example:1.2.3" was not found`)
}

func TestPushCatalogRequiresCachedFromReference(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, err := PushCatalog(context.Background(), PushOptions{
		Reference: "oci://registry.example.com/catalogs/example:1.2.3",
		From:      "oci://registry.example.com/catalogs/example:1.2.3",
	})
	require.Error(t, err)
	assert.EqualError(t, err, `cached catalog "oci://registry.example.com/catalogs/example:1.2.3" was not found`)
}

func TestResolveCachedPushSourceUsesDestinationReferenceWithoutCatalogRoot(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	catalogRoot := writeTestCatalogManifest(t, "example", "1.2.3")
	packaged, err := PackageCatalog(PackageOptions{
		CatalogRoot:   catalogRoot,
		ReferenceBase: "oci://ghcr.io/kubara-io/catalogs/",
	})
	require.NoError(t, err)

	artifact, sourceFrom, err := resolveCachedPushSource(PushOptions{
		Reference: packaged.Reference,
	})
	require.NoError(t, err)
	assert.Equal(t, packaged.Artifact, artifact)
	assert.Equal(t, packaged.Reference, sourceFrom)
}

func writeTestCatalogManifest(t *testing.T, name, version string) string {
	t.Helper()

	root := t.TempDir()
	content := []byte("apiVersion: kubara.io/v1alpha1\nkind: Catalog\nmetadata:\n  name: " + name + "\nspec:\n  version: " + version + "\n")
	require.NoError(t, os.WriteFile(filepath.Join(root, "Catalog.yaml"), content, 0o600))

	return root
}
