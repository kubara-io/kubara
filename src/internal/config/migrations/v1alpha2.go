package migrations

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/rs/zerolog/log"
)

// migrateV1Alpha2Config migrates configurations with version ConfigVersionV1Alpha2 to the ConfigVersionV1Alpha3 schema format,
// moving service catalog directories and renaming additional-values files.
func migrateV1Alpha2Config(cwd string, config map[string]any) error {
	log.Info().Msg("migrating config from v1alpha2 format to v1alpha3")
	log.Info().Msg(`
This migration restructures your repository layout:
  - 'managed-service-catalog' becomes 'platform-components'
  - 'customer-service-catalog' becomes 'platform-configs'
  - The internal directories are refactored from '<tool>/<cluster>' to '<cluster>/<tool>' (e.g. 'helm/my-cluster' -> 'my-cluster/helm')
As a result, your subsequent git changes will look exceptionally large.`)
	config["version"] = ConfigVersionV1Alpha3
	clustersRaw, ok := config["clusters"]
	if !ok {
		return nil
	}

	clusters, ok := clustersRaw.([]any)
	if !ok {
		return nil
	}

	for i, clusterRaw := range clusters {
		cluster, ok := clusterRaw.(map[string]any)
		if !ok {
			continue
		}

		if err := migrateV1Alpha2Cluster(cluster, i); err != nil {
			return fmt.Errorf("cannot migrate cluster number %d: %w", i, err)
		}

		clusterName, ok := cluster["name"].(string)
		if !ok || strings.TrimSpace(clusterName) == "" {
			return fmt.Errorf("%s.name must be a non-empty string", clusterLabel(cluster, i))
		}
		if err := migrateV1Alpha2Files(cwd, clusterName); err != nil {
			return fmt.Errorf("cannot migrate directory structure for cluster %s: %w", clusterLabel(cluster, i), err)
		}

		clusters[i] = cluster
	}

	config["clusters"] = clusters

	managedDir := filepath.Join(cwd, "managed-service-catalog")
	if _, err := os.Stat(managedDir); err == nil {
		if err := moveDirContents(managedDir, filepath.Join(cwd, "platform-components")); err != nil {
			return fmt.Errorf("cannot migrate managed-service-catalog to platform-components: %w", err)
		}
		if err := removeDirIfEmpty(managedDir); err != nil {
			return fmt.Errorf("remove managed-service-catalog dir %q: %w", managedDir, err)
		}
	}
	customerDir := filepath.Join(cwd, "customer-service-catalog")
	if _, err := os.Stat(customerDir); err == nil {
		if err := moveDirContents(customerDir, filepath.Join(cwd, "platform-configs")); err != nil {
			return fmt.Errorf("cannot migrate customer-service-catalog to platform-configs: %w", err)
		}
		if err := removeDirIfEmpty(customerDir); err != nil {
			return fmt.Errorf("remove customer-service-catalog dir %q: %w", customerDir, err)
		}
	}

	return nil
}

func migrateV1Alpha2Cluster(cluster map[string]any, clusterIndex int) error {
	argocd, ok := cluster["argocd"]
	if !ok {
		return nil
	}

	argocdMap, ok := argocd.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.argocd must be an object", clusterLabel(cluster, clusterIndex))
	}

	repo, ok := argocdMap["repo"]
	if !ok {
		return nil
	}

	repoMap, ok := repo.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.argocd.repo must be an object", clusterLabel(cluster, clusterIndex))
	}

	httpsRepo, hasHttpsRepo := repoMap["https"]
	ociRepo, hasOCIRepo := repoMap["oci"]
	if !hasHttpsRepo && !hasOCIRepo {
		return nil
	}

	if hasOCIRepo {
		if err := migrateV1Alpha2Repo(ociRepo); err != nil {
			return fmt.Errorf("cannot migrate OCI repo: %w", err)
		}
	}

	if hasHttpsRepo {
		if err := migrateV1Alpha2Repo(httpsRepo); err != nil {
			return fmt.Errorf("cannot migrate HTTPS repo: %w", err)
		}
	}

	return nil
}

func migrateV1Alpha2Repo(repo any) error {
	repoMap, ok := repo.(map[string]any)
	if !ok {
		return fmt.Errorf("repo must be an object")
	}

	if customer, exists := repoMap["customer"]; exists {
		repoMap["configs"] = customer
		delete(repoMap, "customer")
	}
	if managed, exists := repoMap["managed"]; exists {
		repoMap["components"] = managed
		delete(repoMap, "managed")
	}
	if configs, exists := repoMap["configs"]; exists {
		normalizeRepoPath(configs, "customer-service-catalog/helm", "platform-configs")
		normalizeRepoPath(configs, "platform-configs/helm", "platform-configs")
	}
	if components, exists := repoMap["components"]; exists {
		normalizeRepoPath(components, "managed-service-catalog/helm", "platform-components/helm")
	}
	return nil
}

func normalizeRepoPath(repo any, from, to string) {
	repoMap, ok := repo.(map[string]any)
	if !ok {
		return
	}
	pathValue, ok := repoMap["path"].(string)
	if !ok || pathValue != from {
		return
	}
	repoMap["path"] = to
}

type foundDir struct {
	CatalogDir string // cwd/customer-service-catalog
	SubDir     string // helm, terraform, scripts, ...
	Src        string // cwd/customer-service-catalog/<category>/<clusterName>
}

func migrateV1Alpha2Files(cwd string, clusterName string) error {
	if clusterName == "" {
		return fmt.Errorf("clusterName is empty")
	}

	if err := migrateLegacyValuesFiles(cwd); err != nil {
		return err
	}

	pattern := filepath.Join(cwd, "customer-service-catalog", "*", clusterName)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob %q: %w", pattern, err)
	}

	var found []foundDir

	for _, p := range matches {
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("stat %q: %w", p, err)
		}
		if !info.IsDir() {
			continue
		}

		subDir := filepath.Dir(p)           // .../customer-service-catalog/helm
		catalogDir := filepath.Dir(subDir)  // .../customer-service-catalog
		subDirBase := filepath.Base(subDir) // helm

		found = append(found, foundDir{
			CatalogDir: catalogDir,
			SubDir:     subDirBase,
			Src:        p,
		})
	}

	sort.Slice(found, func(i, j int) bool {
		if found[i].CatalogDir == found[j].CatalogDir {
			return found[i].SubDir < found[j].SubDir
		}
		return found[i].CatalogDir < found[j].CatalogDir
	})

	for _, item := range found {
		dstDir := filepath.Join(item.CatalogDir, clusterName, item.SubDir)

		if err := moveDirContents(item.Src, dstDir); err != nil {
			return err
		}

		if err := os.Remove(item.Src); err != nil {
			return fmt.Errorf("remove empty dir %q: %w", item.Src, err)
		}

		if err := removeDirIfEmpty(filepath.Join(item.CatalogDir, item.SubDir)); err != nil {
			return err
		}
	}

	return nil
}

func migrateLegacyValuesFiles(cwd string) (err error) {
	for _, root := range []struct {
		path             string
		renameLegacyHelm bool
	}{
		{path: filepath.Join(cwd, "customer-service-catalog"), renameLegacyHelm: true},
		{path: filepath.Join(cwd, "platform-configs")},
		{path: filepath.Join(cwd, "managed-service-catalog")},
		{path: filepath.Join(cwd, "platform-components")},
	} {
		info, statErr := os.Stat(root.path)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return fmt.Errorf("stat scoped migration root %q: %w", root.path, statErr)
		}
		if !info.IsDir() {
			continue
		}

		if err := migrateLegacyValuesFilesInRoot(root.path, root.renameLegacyHelm); err != nil {
			return err
		}
	}

	return nil
}

func migrateLegacyValuesFilesInRoot(rootPath string, renameLegacyHelm bool) (err error) {
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return fmt.Errorf("open root %q: %w", rootPath, err)
	}
	defer func() {
		closeErr := root.Close()
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close root %q: %w", rootPath, closeErr)
		}
	}()

	var renameAdditional []string
	var renameLegacyValues []string

	if err := fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		switch filepath.Base(path) {
		case "additional-values.yaml":
			renameAdditional = append(renameAdditional, path)
		case "values.yaml":
			if !renameLegacyHelm || !strings.HasPrefix(filepath.ToSlash(path), "helm/") {
				return nil
			}
			serviceDir := filepath.Base(filepath.Dir(path))
			if hasGeneratedValuesTemplate(serviceDir) {
				renameLegacyValues = append(renameLegacyValues, path)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("walk %q: %w", rootPath, err)
	}

	for _, path := range renameAdditional {
		newPath := filepath.Join(filepath.Dir(path), "values-additional.yaml")
		if err := root.Rename(path, newPath); err != nil {
			return fmt.Errorf("rename %q to %q: %w", path, newPath, err)
		}
	}

	for _, path := range renameLegacyValues {
		newPath := filepath.Join(filepath.Dir(path), "values.generated.yaml")
		if err := root.Rename(path, newPath); err != nil {
			return fmt.Errorf("rename %q: %w", path, err)
		}
	}

	return nil
}

func hasGeneratedValuesTemplate(serviceDir string) bool {
	tpltPath := filepath.ToSlash(filepath.Join("built-in", "platform-configs", "helm", serviceDir, "values.generated.yaml.tplt"))
	_, err := fs.Stat(catalog.BuiltInFS(), tpltPath)
	return err == nil
}
