package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

type SourceKind string

const (
	SourceKindDirectory SourceKind = "directory"
	SourceKindOCI       SourceKind = "oci"
)

type ResolvedSource struct {
	Kind         SourceKind
	RootPath     string
	ServicesPath string
	Manifest     *CatalogManifest
	Artifact     *CachedArtifact
}

func ResolveSource(catalogPath string) (ResolvedSource, error) {
	raw := strings.TrimSpace(catalogPath)
	if raw == "" {
		return ResolvedSource{}, nil
	}

	if !IsOCIReference(raw) {
		return resolveDirectorySource(raw)
	}

	if _, err := ParseOCIReference(raw); err != nil {
		return ResolvedSource{}, err
	}

	// TODO ensure we pull missing catalogs from remote
	artifact, err := GetCachedArtifact(raw)
	if err != nil {
		return ResolvedSource{}, err
	}

	return resolvedSourceFromArtifact(artifact)
}

func requireDirectory(path, label string) (string, error) {
	cleaned := filepath.Clean(path)
	info, err := os.Stat(cleaned)
	if err != nil {
		return "", fmt.Errorf("%s %q does not exist: %w", label, cleaned, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s %q is not a directory", label, cleaned)
	}
	return cleaned, nil
}

func resolveDirectorySource(catalogPath string) (ResolvedSource, error) {
	cleaned, err := requireDirectory(catalogPath, "catalog path")
	if err != nil {
		return ResolvedSource{}, err
	}
	servicesPath, err := resolveServicesPath(cleaned)
	if err != nil {
		return ResolvedSource{}, err
	}

	source := ResolvedSource{
		Kind:         SourceKindDirectory,
		RootPath:     cleaned,
		ServicesPath: servicesPath,
	}

	manifest, err := tryLoadCatalogManifest(cleaned)
	if err != nil {
		return ResolvedSource{}, err
	}
	source.Manifest = manifest

	return source, nil
}

func resolvedSourceFromArtifact(artifact CachedArtifact) (ResolvedSource, error) {
	rootPath := artifactRootPath(artifact)
	servicesPath, err := resolveServicesPath(rootPath)
	if err != nil {
		return ResolvedSource{}, err
	}

	manifest, err := LoadCatalogManifest(rootPath)
	if err != nil {
		return ResolvedSource{}, err
	}

	return ResolvedSource{
		Kind:         SourceKindOCI,
		RootPath:     rootPath,
		ServicesPath: servicesPath,
		Manifest:     &manifest,
		Artifact:     &artifact,
	}, nil
}

func LoadCatalogManifest(root string) (CatalogManifest, error) {
	path := filepath.Join(root, "Catalog.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		return CatalogManifest{}, fmt.Errorf("read %q: %w", path, err)
	}

	var manifest CatalogManifest
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		return CatalogManifest{}, fmt.Errorf("unmarshal %q: %w", path, err)
	}
	if err := manifest.Validate(); err != nil {
		return CatalogManifest{}, fmt.Errorf("invalid Catalog.yaml: %w", err)
	}

	return manifest, nil
}

func tryLoadCatalogManifest(root string) (*CatalogManifest, error) {
	path := filepath.Join(root, "Catalog.yaml")
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %q: %w", path, err)
	}

	manifest, err := LoadCatalogManifest(root)
	if err != nil {
		return nil, err
	}
	return &manifest, nil
}

func resolveServicesPath(catalogPath string) (string, error) {
	cleaned, err := requireDirectory(catalogPath, "catalog path")
	if err != nil {
		return "", err
	}

	servicesDir := filepath.Join(cleaned, string(ServicesDirectory))
	return requireDirectory(servicesDir, "catalog services path")
}
