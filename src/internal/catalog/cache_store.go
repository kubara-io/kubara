package catalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
)

type UnpackageOptions struct {
	Reference  string
	OutputPath string
	WorkDir    string
}

func defaultCatalogCacheRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(home, defaultCatalogCacheRel), nil
}

func newTempArtifactDir() (string, func(), error) {
	cacheRoot, err := defaultCatalogCacheRoot()
	if err != nil {
		return "", nil, err
	}
	tempRoot := filepath.Join(cacheRoot, ".tmp")
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return "", nil, fmt.Errorf("create temporary catalog cache directory: %w", err)
	}
	tempArtifactDir, err := os.MkdirTemp(tempRoot, "artifact-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temporary artifact directory: %w", err)
	}
	return tempArtifactDir, func() { _ = os.RemoveAll(tempArtifactDir) }, nil
}

func finalizeCachedArtifact(tempArtifactDir string, artifact CachedArtifact) error {
	finalDir, err := artifactDirPath(artifact.ManifestDigest)
	if err != nil {
		return err
	}

	if err := writeArtifactMetadata(filepath.Join(tempArtifactDir, "metadata.json"), artifact); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(finalDir), 0o755); err != nil {
		return fmt.Errorf("create catalog artifact cache directory: %w", err)
	}
	if _, err := os.Stat(finalDir); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat catalog artifact cache directory %q: %w", finalDir, err)
	}

	if err := os.Rename(tempArtifactDir, finalDir); err != nil {
		if _, statErr := os.Stat(finalDir); statErr == nil {
			return nil
		}
		return fmt.Errorf("move catalog artifact into cache: %w", err)
	}
	return nil
}

func readJSONFile(path, label string, target any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s %q: %w", label, path, err)
	}
	if err := json.Unmarshal(content, target); err != nil {
		return fmt.Errorf("unmarshal %s %q: %w", label, path, err)
	}
	return nil
}

func writeJSONFile(path, label string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", label, err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", label, err)
	}
	return nil
}

func writeArtifactMetadata(path string, artifact CachedArtifact) error {
	return writeJSONFile(path, "cached artifact metadata", artifact)
}

func artifactDirPath(manifestDigest string) (string, error) {
	cacheRoot, err := defaultCatalogCacheRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheRoot, "artifacts", digestPathComponent(manifestDigest)), nil
}

func artifactRootPath(artifact CachedArtifact) string {
	dir, _ := artifactDirPath(artifact.ManifestDigest)
	return filepath.Join(dir, "contents", artifact.RootDirectory)
}

func findArtifactByDigest(dgst digest.Digest) (CachedArtifact, bool, error) {
	path, err := artifactDirPath(dgst.String())
	if err != nil {
		return CachedArtifact{}, false, err
	}
	artifact, found, err := readArtifactMetadata(path)
	if err != nil {
		return CachedArtifact{}, false, err
	}
	return artifact, found, nil
}

func findTagArtifact(ref OCIReference) (CachedArtifact, bool, error) {
	refPath, err := tagReferencePath(ref)
	if err != nil {
		return CachedArtifact{}, false, err
	}
	cachedRef, found, err := readCachedReferenceFile(refPath)
	if err != nil || !found {
		return CachedArtifact{}, false, err
	}

	artifact, found, err := findArtifactByDigest(digest.Digest(cachedRef.ManifestDigest))
	if err != nil || !found {
		return CachedArtifact{}, false, err
	}
	return artifact, true, nil
}

func readArtifactMetadata(artifactDir string) (CachedArtifact, bool, error) {
	path := filepath.Join(artifactDir, "metadata.json")
	var artifact CachedArtifact
	if err := readJSONFile(path, "cached artifact metadata", &artifact); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CachedArtifact{}, false, nil
		}
		return CachedArtifact{}, false, err
	}
	if strings.TrimSpace(artifact.SchemaVersion) == "" {
		return CachedArtifact{}, false, fmt.Errorf("cached artifact metadata %q is missing schemaVersion", path)
	}
	if strings.TrimSpace(artifact.CatalogName) == "" {
		return CachedArtifact{}, false, fmt.Errorf("cached artifact metadata %q is missing catalogName", path)
	}
	if strings.TrimSpace(artifact.CatalogVersion) == "" {
		return CachedArtifact{}, false, fmt.Errorf("cached artifact metadata %q is missing catalogVersion", path)
	}
	if strings.TrimSpace(artifact.ManifestDigest) == "" {
		return CachedArtifact{}, false, fmt.Errorf("cached artifact metadata %q is missing manifestDigest", path)
	}
	if strings.TrimSpace(artifact.RootDirectory) == "" {
		return CachedArtifact{}, false, fmt.Errorf("cached artifact metadata %q is missing rootDirectory", path)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "layout", "index.json")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CachedArtifact{}, false, nil
		}
		return CachedArtifact{}, false, fmt.Errorf("stat cached OCI layout for %q: %w", artifactDir, err)
	}
	if _, err := os.Stat(filepath.Join(artifactDir, "contents", artifact.RootDirectory, "Catalog.yaml")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return CachedArtifact{}, false, nil
		}
		return CachedArtifact{}, false, fmt.Errorf("stat cached catalog contents for %q: %w", artifactDir, err)
	}

	return artifact, true, nil
}

func writeLocalReference(ref OCIReference, artifact CachedArtifact) error {
	refPath, err := tagReferencePath(ref)
	if err != nil {
		return err
	}
	return writeReferenceFile(refPath, cachedReference{
		SchemaVersion:  cacheSchemaVersion,
		ManifestDigest: artifact.ManifestDigest,
		CatalogName:    artifact.CatalogName,
		CatalogVersion: artifact.CatalogVersion,
		Reference:      ref.Raw,
		Registry:       ref.Registry,
		Repository:     ref.Repository,
		Tag:            ref.Tag,
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	})
}

func writeReferenceFile(path string, ref cachedReference) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create cached catalog reference directory: %w", err)
	}
	return writeJSONFile(path, "cached catalog reference", ref)
}

func tagReferencePath(ref OCIReference) (string, error) {
	cacheRoot, err := defaultCatalogCacheRoot()
	if err != nil {
		return "", err
	}

	repositoryParts := strings.Split(ref.Repository, "/")
	pathParts := []string{cacheRoot, "refs", sanitizePathComponent(ref.Registry)}
	for _, part := range repositoryParts {
		pathParts = append(pathParts, sanitizePathComponent(part))
	}
	pathParts = append(pathParts, "tags", ref.Tag+".json")
	return filepath.Join(pathParts...), nil
}

func sanitizePathComponent(value string) string {
	replacer := strings.NewReplacer(":", "_", "@", "_", "\\", "_")
	return replacer.Replace(value)
}

func digestPathComponent(manifestDigest string) string {
	return strings.ReplaceAll(manifestDigest, ":", "-")
}

func isLocalhostReference(ref OCIReference) bool {
	return ref.Registry == "localhost"
}
