package config

import (
	"os"
	"path/filepath"

	"github.com/kubara-io/kubara/internal/catalog"
	testutilpkg "github.com/kubara-io/kubara/internal/testutil"
)

var (
	testBootstrapCatalogPath string
	testGeneralCatalogPath   string
	testCustomCatalogPath    string
)

func init() {
	root, err := os.MkdirTemp("", "kubara-config-catalog-tests-*")
	if err != nil {
		panic(err)
	}

	bootstrapPath, generalPath, err := testutilpkg.CreateCatalogFixtures(filepath.Join(root, "catalogs"))
	if err != nil {
		panic(err)
	}
	customPath := filepath.Join(root, "catalogs", "custom-catalog")
	if err := os.MkdirAll(filepath.Join(customPath, "services"), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(customPath, "Catalog.yaml"), []byte(`apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: custom
spec:
  version: 1.0.0
`), 0o600); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(customPath, "services", "loki.yaml"), []byte(`apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: loki
spec:
  chartPath: loki
  status: enabled
`), 0o600); err != nil {
		panic(err)
	}

	testBootstrapCatalogPath = bootstrapPath
	testGeneralCatalogPath = generalPath
	testCustomCatalogPath = customPath
}

func testCatalogLoadOptions() catalog.LoadOptions {
	return catalog.LoadOptions{
		BootstrapCatalog: testBootstrapCatalogPath,
		Catalogs:         []string{testGeneralCatalogPath},
	}
}

func testClusterCatalogs() []string {
	return []string{testGeneralCatalogPath}
}

func testBootstrapCatalogPtr() *string {
	path := testBootstrapCatalogPath
	return &path
}
