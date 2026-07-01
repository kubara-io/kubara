package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/kubara-io/kubara/internal/catalog"
)

func applyMigrations(cwd string, config map[string]any) (bool, error) {
	var migrated bool
	if isLegacyConfig(config) {
		if err := migrateLegacyConfig(config); err != nil {
			return false, fmt.Errorf("migrate legacy config: %w", err)
		}
		migrated = true
	}

	if isV1Alpha1Config(config) {
		if err := migrateV1Alpha1Config(config); err != nil {
			return false, fmt.Errorf("migrate V1Alpha1 config: %w", err)
		}
		migrated = true
	}

	if isV1Alpha2Config(config) {
		if err := migrateV1Alpha2Config(cwd, config); err != nil {
			return false, fmt.Errorf("migrate V1Alpha2 config: %w", err)
		}
		migrated = true
	}

	return migrated, nil
}

func isLegacyConfig(raw map[string]any) bool {
	_, hasVersion := raw["version"]
	return !hasVersion
}

func isV1Alpha1Config(raw map[string]any) bool {
	version, hasVersion := raw["version"]
	return version == ConfigVersionV1Alpha1 && hasVersion
}

func isV1Alpha2Config(raw map[string]any) bool {
	version, hasVersion := raw["version"]
	return version == ConfigVersionV1Alpha2 && hasVersion
}

func migrateLegacyConfig(raw map[string]any) error {
	clustersRaw, ok := raw["clusters"]
	if !ok {
		raw["version"] = ConfigVersionV1Alpha1
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

		if err := migrateLegacyCluster(cluster, i); err != nil {
			return fmt.Errorf("cannot migrate cluster number %d: %w", i, err)
		}

		clusters[i] = cluster
	}

	raw["clusters"] = clusters
	raw["version"] = ConfigVersionV1Alpha1
	return nil
}

func migrateV1Alpha1Config(config map[string]any) error {
	config["version"] = ConfigVersionV1Alpha2
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

		if err := migrateV1Alpha1Cluster(cluster, i); err != nil {
			return fmt.Errorf("cannot migrate cluster number %d: %w", i, err)
		}

		clusters[i] = cluster
	}

	config["clusters"] = clusters
	return nil
}

func migrateV1Alpha2Config(cwd string, config map[string]any) error {
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

func removeDirIfEmpty(dir string) error {
	if err := os.Remove(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTEMPTY) {
			return nil
		}

		return fmt.Errorf("remove empty dir %q: %w", dir, err)
	}

	return nil
}

func moveDirContents(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("readdir %q: %w", srcDir, err)
	}

	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %q: %w", dstDir, err)
	}

	for _, entry := range entries {
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())

		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("destination already exists: %q", dst)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat %q: %w", dst, err)
		}

		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("move %q -> %q: %w", src, dst, err)
		}
	}

	return nil
}

func migrateV1Alpha1Cluster(cluster map[string]any, clusterIndex int) error {
	publicIps, hasPublic := cluster["publicLoadBalancerIP"]
	privateIps, hasPrivate := cluster["privateLoadBalancerIP"]
	if !hasPublic && !hasPrivate {
		return nil
	}

	servicesRaw, ok := cluster["services"]
	if !ok {
		return nil
	}

	servicesMap, ok := servicesRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.services must be an object", clusterLabel(cluster, clusterIndex))
	}

	metallb, ok := servicesMap["metallb"].(map[string]any)
	if !ok {
		metallb = map[string]any{}
	}

	metallb["config"] = map[string]any{
		"publicLoadBalancerIPs":   publicIps,
		"loadBalancerAddressPool": []any{fmt.Sprintf("%s/32", privateIps)},
	}
	servicesMap["metallb"] = metallb
	delete(cluster, "publicLoadBalancerIP")
	delete(cluster, "privateLoadBalancerIP")
	return nil
}

func migrateLegacyCluster(cluster map[string]any, clusterIndex int) error {
	clusterTypeRaw := cluster["type"]
	if clusterTypeRaw != nil {
		clusterType, ok := clusterTypeRaw.(string)
		if !ok {
			return fmt.Errorf("cluster.type must be a string")
		}

		switch clusterType {
		case "worker":
			cluster["type"] = "spoke"
		default:
			cluster["type"] = "hub"
		}
	}

	servicesRaw, ok := cluster["services"]
	if !ok {
		return nil
	}

	servicesMap, ok := servicesRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.services must be an object", clusterLabel(cluster, clusterIndex))
	}

	serviceContext := clusterLabel(cluster, clusterIndex)
	migratedServices := make(map[string]any, len(servicesMap))
	sourceByCanonical := make(map[string]string, len(servicesMap))

	for originalName, serviceRaw := range servicesMap {
		canonicalName := catalog.CanonicalServiceName(originalName)
		if previousName, exists := sourceByCanonical[canonicalName]; exists {
			return fmt.Errorf("%s.services has conflicting keys %q and %q for canonical service %q", serviceContext, previousName, originalName, canonicalName)
		}

		if serviceMap, ok := serviceRaw.(map[string]any); ok {
			if err := migrateLegacyService(canonicalName, serviceMap, serviceContext); err != nil {
				return err
			}
			migratedServices[canonicalName] = serviceMap
		} else {
			migratedServices[canonicalName] = serviceRaw
		}

		sourceByCanonical[canonicalName] = originalName
	}

	cluster["services"] = migratedServices
	return nil
}

func migrateLegacyService(serviceName string, serviceMap map[string]any, clusterContext string) error {
	serviceContext := fmt.Sprintf("%s.services.%s", clusterContext, serviceName)

	if serviceName == "cert-manager" {
		if err := migrateLegacyClusterIssuer(serviceMap, serviceContext); err != nil {
			return err
		}
	}

	switch serviceName {
	case "kube-prometheus-stack", "loki":
		if err := migrateLegacyStorageClassName(serviceMap, serviceContext); err != nil {
			return err
		}
	}

	if err := migrateLegacyIngressAnnotations(serviceMap, serviceContext); err != nil {
		return err
	}

	return nil
}

func migrateLegacyClusterIssuer(serviceMap map[string]any, serviceContext string) error {
	clusterIssuer, ok := serviceMap["clusterIssuer"]
	if !ok {
		return nil
	}

	configMap, err := ensureNestedObject(serviceMap, "config", serviceContext)
	if err != nil {
		return err
	}
	if _, exists := configMap["clusterIssuer"]; exists {
		return fmt.Errorf("%s has both legacy clusterIssuer and config.clusterIssuer", serviceContext)
	}

	configMap["clusterIssuer"] = clusterIssuer
	delete(serviceMap, "clusterIssuer")
	return nil
}

func migrateLegacyStorageClassName(serviceMap map[string]any, serviceContext string) error {
	storageClassName, ok := serviceMap["storageClassName"]
	if !ok {
		return nil
	}

	storageMap, err := ensureNestedObject(serviceMap, "storage", serviceContext)
	if err != nil {
		return err
	}
	if _, exists := storageMap["className"]; exists {
		return fmt.Errorf("%s has both legacy storageClassName and storage.className", serviceContext)
	}

	storageMap["className"] = storageClassName
	delete(serviceMap, "storageClassName")
	return nil
}

func migrateLegacyIngressAnnotations(serviceMap map[string]any, serviceContext string) error {
	ingressRaw, ok := serviceMap["ingress"]
	if !ok {
		return nil
	}

	ingressMap, ok := ingressRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.ingress must be an object", serviceContext)
	}

	annotations, ok := ingressMap["annotations"]
	if !ok {
		return nil
	}

	networkingMap, err := ensureNestedObject(serviceMap, "networking", serviceContext)
	if err != nil {
		return err
	}
	if _, exists := networkingMap["annotations"]; exists {
		return fmt.Errorf("%s has both legacy ingress.annotations and networking.annotations", serviceContext)
	}

	networkingMap["annotations"] = annotations
	delete(ingressMap, "annotations")
	if len(ingressMap) == 0 {
		delete(serviceMap, "ingress")
	} else {
		serviceMap["ingress"] = ingressMap
	}

	return nil
}

func ensureNestedObject(parent map[string]any, key, context string) (map[string]any, error) {
	raw, exists := parent[key]
	if !exists || raw == nil {
		nested := map[string]any{}
		parent[key] = nested
		return nested, nil
	}

	nested, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s.%s must be an object", context, key)
	}

	return nested, nil
}

func clusterLabel(cluster map[string]any, clusterIndex int) string {
	if name, ok := cluster["name"].(string); ok && strings.TrimSpace(name) != "" {
		return fmt.Sprintf("cluster %q", name)
	}

	return fmt.Sprintf("clusters[%d]", clusterIndex)
}
