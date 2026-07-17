package catalog

import (
	"fmt"
	"strings"

	"github.com/kubara-io/kubara/internal/utils"
)

func ResolveLoadOptions(cwd string, catalogs []string, overwrite bool) (LoadOptions, error) {
	if len(catalogs) == 0 {
		return LoadOptions{}, fmt.Errorf("no catalog provided")
	}

	resolvedCatalogs := make([]string, 0, len(catalogs))
	for _, cat := range catalogs {
		if strings.TrimSpace(cat) == "" {
			return LoadOptions{}, fmt.Errorf("catalog source is empty")
		}

		if IsOCIReference(cat) {
			resolvedCatalogs = append(resolvedCatalogs, cat)
		} else {
			absolutePath, err := utils.GetFullPath(cat, cwd)
			if err != nil {
				return LoadOptions{}, fmt.Errorf("get catalog path: %w", err)
			}
			resolvedCatalogs = append(resolvedCatalogs, absolutePath)
		}
	}

	return LoadOptions{
		Catalogs:  resolvedCatalogs,
		Overwrite: overwrite,
	}, nil
}
