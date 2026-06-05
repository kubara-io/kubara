package catalog

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kubara-io/kubara/internal/utils"
)

type CachedCatalogEntry struct {
	Reference      string
	CatalogName    string
	CatalogVersion string
	ManifestDigest string
}

type UnpackageResult struct {
	Artifact   CachedArtifact
	OutputPath string
}

func ListCachedCatalogs() ([]CachedCatalogEntry, error) {
	cacheRoot, err := defaultCatalogCacheRoot()
	if err != nil {
		return nil, err
	}

	entries := make(map[string]CachedCatalogEntry)
	referencedDigests := make(map[string]struct{})

	if err := collectReferenceEntries(filepath.Join(cacheRoot, "refs"), entries, referencedDigests); err != nil {
		return nil, err
	}

	result := make([]CachedCatalogEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry)
	}

	// Sort by name, then version and then source reference
	slices.SortFunc(result, func(a, b CachedCatalogEntry) int {
		if cmp := strings.Compare(a.CatalogName, b.CatalogName); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.CatalogVersion, b.CatalogVersion); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Reference, b.Reference)
	})

	return result, nil
}

func UnpackageCatalog(options UnpackageOptions) (UnpackageResult, error) {
	artifact, err := GetCachedArtifact(options.Reference)
	if err != nil {
		return UnpackageResult{}, err
	}

	outputPath := strings.TrimSpace(options.OutputPath)
	if outputPath == "" {
		outputPath, err = utils.GetFullPath(artifact.CatalogName, options.WorkDir)
		if err != nil {
			return UnpackageResult{}, fmt.Errorf("resolve default unpack directory: %w", err)
		}
	}

	if _, err := os.Stat(outputPath); err == nil {
		return UnpackageResult{}, fmt.Errorf("catalog output directory %q already exists", outputPath)
	} else if !os.IsNotExist(err) {
		return UnpackageResult{}, fmt.Errorf("stat catalog output directory %q: %w", outputPath, err)
	}

	if err := copyDirectory(artifactRootPath(artifact), outputPath); err != nil {
		return UnpackageResult{}, err
	}

	return UnpackageResult{
		Artifact:   artifact,
		OutputPath: outputPath,
	}, nil
}

func collectReferenceEntries(root string, entries map[string]CachedCatalogEntry, referencedDigests map[string]struct{}) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("walk cached catalog references: %w", err)
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		ref, err := readCachedReference(path)
		if err != nil {
			return err
		}

		entry := CachedCatalogEntry{
			Reference:      firstNonEmpty(ref.Reference, remoteReferenceFromCachedReference(ref), defaultLocalCatalogReference(ref.CatalogName, ref.CatalogVersion), fmt.Sprintf("%s:%s", ref.CatalogName, ref.CatalogVersion)),
			CatalogName:    ref.CatalogName,
			CatalogVersion: ref.CatalogVersion,
			ManifestDigest: ref.ManifestDigest,
		}
		entries[entry.Reference] = entry
		referencedDigests[entry.ManifestDigest] = struct{}{}
		return nil
	})
}

func readCachedReference(path string) (cachedReference, error) {
	var ref cachedReference
	if err := readJSONFile(path, "cached catalog reference", &ref); err != nil {
		return cachedReference{}, err
	}
	return ref, nil
}

func readCachedReferenceFile(path string) (cachedReference, bool, error) {
	ref, err := readCachedReference(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cachedReference{}, false, nil
		}
		return cachedReference{}, false, err
	}
	return ref, true, nil
}

func remoteReferenceFromCachedReference(ref cachedReference) string {
	if ref.Registry == "" || ref.Repository == "" || ref.Tag == "" {
		return ref.ManifestDigest
	}
	return fmt.Sprintf("oci://%s/%s:%s", ref.Registry, ref.Repository, ref.Tag)
}

func copyDirectory(sourceRoot, targetRoot string) error {
	return filepath.WalkDir(sourceRoot, func(sourcePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk cached catalog contents %q: %w", sourcePath, err)
		}

		relativePath, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil {
			return fmt.Errorf("build relative path for %q: %w", sourcePath, err)
		}
		targetPath := filepath.Join(targetRoot, relativePath)

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read file info for %q: %w", sourcePath, err)
		}

		switch {
		case entry.IsDir():
			if err := os.MkdirAll(targetPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("create catalog output directory %q: %w", targetPath, err)
			}
			return nil
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("cached catalog contains unsupported symlink %q", sourcePath)
		case !info.Mode().IsRegular():
			return fmt.Errorf("cached catalog contains unsupported file type %q", sourcePath)
		default:
			return copyFile(sourcePath, targetPath, info.Mode().Perm())
		}
	})
}

func copyFile(sourcePath, targetPath string, mode fs.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open cached catalog file %q: %w", sourcePath, err)
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("create catalog output file %q: %w", targetPath, err)
	}

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		if closeErr := targetFile.Close(); closeErr != nil {
			return errors.Join(
				fmt.Errorf("copy catalog file %q: %w", targetPath, err),
				fmt.Errorf("close catalog output file %q after copy failure: %w", targetPath, closeErr),
			)
		}
		return fmt.Errorf("copy catalog file %q: %w", targetPath, err)
	}
	if err := targetFile.Close(); err != nil {
		return fmt.Errorf("close catalog output file %q: %w", targetPath, err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func defaultLocalCatalogReference(catalogName, version string) string {
	ref, err := BuildCatalogReference(catalogName, version, "")
	if err != nil {
		return ""
	}
	return ref.Raw
}
