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

	testBootstrapCatalogPath = bootstrapPath
	testGeneralCatalogPath = generalPath
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
