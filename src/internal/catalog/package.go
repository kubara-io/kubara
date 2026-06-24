package catalog

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/content/oci"
)

type PackageResult struct {
	Manifest  CatalogManifest
	Artifact  CachedArtifact
	Reference string
}

type PackageOptions struct {
	CatalogRoot   string
	ReferenceBase string
}

func PackageCatalog(options PackageOptions) (PackageResult, error) {
	manifest, err := LoadCatalogManifest(options.CatalogRoot)
	if err != nil {
		return PackageResult{}, err
	}

	ref, err := BuildCatalogReference(manifest.Metadata.Name, manifest.Spec.Version, options.ReferenceBase)
	if err != nil {
		return PackageResult{}, err
	}

	artifact, err := createCachedArtifact(manifest, options.CatalogRoot, CachedArtifact{
		SchemaVersion:  cacheSchemaVersion,
		CatalogName:    manifest.Metadata.Name,
		CatalogVersion: manifest.Spec.Version,
		RootDirectory:  manifest.Metadata.Name,
	})
	if err != nil {
		return PackageResult{}, err
	}

	if err := writeCachedReference(ref, artifact); err != nil {
		return PackageResult{}, err
	}

	return PackageResult{
		Manifest:  manifest,
		Artifact:  artifact,
		Reference: ref.Raw,
	}, nil
}

func createCachedArtifact(manifest CatalogManifest, catalogRoot string, artifact CachedArtifact) (CachedArtifact, error) {
	tempArtifactDir, cleanup, err := newTempArtifactDir()
	if err != nil {
		return CachedArtifact{}, err
	}
	defer cleanup()

	ctx := context.Background()
	fileStore, err := file.New(tempArtifactDir)
	if err != nil {
		return CachedArtifact{}, fmt.Errorf("create file store: %w", err)
	}
	defer func() {
		_ = fileStore.Close()
	}()
	fileStore.TarReproducible = true

	layerDescriptor, err := fileStore.Add(ctx, manifest.Metadata.Name, CatalogLayerMediaType, catalogRoot)
	if err != nil {
		return CachedArtifact{}, fmt.Errorf("package catalog directory: %w", err)
	}

	packOptions := oras.PackManifestOptions{
		Layers:              []v1.Descriptor{layerDescriptor},
		ManifestAnnotations: buildCatalogManifestAnnotations(manifest),
	}
	manifestDescriptor, err := oras.PackManifest(ctx, fileStore, oras.PackManifestVersion1_1, CatalogArtifactType, packOptions)
	if err != nil {
		return CachedArtifact{}, fmt.Errorf("pack catalog manifest: %w", err)
	}

	if err := fileStore.Tag(ctx, manifestDescriptor, manifest.Spec.Version); err != nil {
		return CachedArtifact{}, fmt.Errorf("tag packaged catalog: %w", err)
	}

	layoutDir := filepath.Join(tempArtifactDir, "layout")
	layoutStore, err := oci.New(layoutDir)
	if err != nil {
		return CachedArtifact{}, fmt.Errorf("create OCI layout store: %w", err)
	}
	if err := oras.CopyGraph(ctx, fileStore, layoutStore, manifestDescriptor, oras.DefaultCopyGraphOptions); err != nil {
		return CachedArtifact{}, fmt.Errorf("copy packaged catalog into OCI layout: %w", err)
	}
	if err := layoutStore.Tag(ctx, manifestDescriptor, manifest.Spec.Version); err != nil {
		return CachedArtifact{}, fmt.Errorf("tag OCI layout with %q: %w", manifest.Spec.Version, err)
	}

	contentsDir := filepath.Join(tempArtifactDir, "contents")
	rootDir, err := extractCatalogContents(ctx, layoutStore, manifestDescriptor, contentsDir)
	if err != nil {
		return CachedArtifact{}, err
	}

	artifact.ManifestDigest = manifestDescriptor.Digest.String()
	artifact.RootDirectory = filepath.Base(rootDir)

	if err := finalizeCachedArtifact(tempArtifactDir, artifact); err != nil {
		return CachedArtifact{}, err
	}

	return artifact, nil
}

func buildCatalogManifestAnnotations(manifest CatalogManifest) map[string]string {
	annotations := maps.Clone(manifest.Metadata.Annotations)
	if annotations == nil {
		annotations = make(map[string]string, 2)
	}

	annotations["io.kubara.catalog.name"] = manifest.Metadata.Name
	annotations["io.kubara.catalog.version"] = manifest.Spec.Version
	// Fake timestamp for forcing immutable manifest digests for the same catalog contents
	annotations[v1.AnnotationCreated] = "1970-01-01T00:00:00Z"

	return annotations
}

func extractCatalogContents(ctx context.Context, layoutStore *oci.Store, desc v1.Descriptor, contentsDir string) (string, error) {
	if err := os.MkdirAll(contentsDir, 0o755); err != nil {
		return "", fmt.Errorf("create extracted catalog directory: %w", err)
	}

	fileStore, err := file.New(contentsDir)
	if err != nil {
		return "", fmt.Errorf("create extracted file store: %w", err)
	}
	defer func() {
		_ = fileStore.Close()
	}()

	if err := oras.CopyGraph(ctx, layoutStore, fileStore, desc, oras.DefaultCopyGraphOptions); err != nil {
		return "", fmt.Errorf("extract catalog contents: %w", err)
	}

	entries, err := os.ReadDir(contentsDir)
	if err != nil {
		return "", fmt.Errorf("read extracted catalog directory: %w", err)
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return "", fmt.Errorf("expected packaged catalog to extract into a single root directory")
	}

	return filepath.Join(contentsDir, entries[0].Name()), nil
}
