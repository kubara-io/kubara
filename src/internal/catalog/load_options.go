package catalog

import (
	"fmt"

	"github.com/kubara-io/kubara/internal/utils"
)

func ResolveLoadOptions(cwd, rawCatalogPath string, overwrite bool) (LoadOptions, error) {
	if rawCatalogPath == "" {
		return LoadOptions{
			CatalogPath: "",
			Overwrite:   overwrite,
		}, nil
	}

	if IsOCIReference(rawCatalogPath) {
		return LoadOptions{
			CatalogPath: rawCatalogPath,
			Overwrite:   overwrite,
		}, nil
	}

	absoluteCatalogPath, err := utils.GetFullPath(rawCatalogPath, cwd)
	if err != nil {
		return LoadOptions{}, fmt.Errorf("get catalog path: %w", err)
	}

	return LoadOptions{
		CatalogPath: absoluteCatalogPath,
		Overwrite:   overwrite,
	}, nil
}
