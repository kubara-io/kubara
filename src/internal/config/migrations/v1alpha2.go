package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	return nil
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

	// Rename any additional-values.yaml to values-additional.yaml, and remove old values.yaml if a matching template was migrated to values.generated.yaml.tplt
	_ = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			base := filepath.Base(path)
			if base == "additional-values.yaml" {
				dir := filepath.Dir(path)
				newPath := filepath.Join(dir, "values-additional.yaml")
				_ = os.Rename(path, newPath)
			} else if base == "values.yaml" {
				dir := filepath.Dir(path)
				serviceDir := filepath.Base(dir) // e.g., argo-cd, loki
				// Find the values.generated.yaml.tplt dynamically relative to go.mod or catalog assets
				// without absolute fallback paths
				for _, rootDir := range []string{cwd, "/workspace", ".", "../.."} {
					for _, relativeSub := range []string{
						"internal/catalog/built-in/platform-configs/helm",
						"catalog/built-in/platform-configs/helm",
					} {
						tpltPath := filepath.Join(rootDir, relativeSub, serviceDir, "values.generated.yaml.tplt")
						if _, statErr := os.Stat(tpltPath); statErr == nil {
							_ = os.Remove(path)
							return nil
						}
					}
				}
			}
		}
		return nil
	})

	pattern := filepath.Join(cwd, "*", "*", clusterName)
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
