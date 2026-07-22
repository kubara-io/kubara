package catalog

import (
	"fmt"
	"strings"

	"github.com/kubara-io/kubara/internal/utils"
)

func ResolveLoadOptions(cwd string, catalogs []string, overwrite bool) (LoadOptions, error) {
	options := LoadOptions{
		CWD:       cwd,
		Catalogs:  append([]string(nil), catalogs...),
		Overwrite: overwrite,
	}
	if _, err := ResolveSources(options); err != nil {
		return LoadOptions{}, err
	}
	return options, nil
}

func ResolveSources(options LoadOptions) ([]string, error) {
	bootstrap := strings.TrimSpace(options.BootstrapCatalog)
	if bootstrap == "" {
		bootstrap = DefaultBootstrapCatalog
	}

	references := append([]string{bootstrap}, options.Catalogs...)
	sources := make([]string, 0, len(references))
	seen := make(map[string]struct{}, len(references))
	for _, reference := range references {
		reference = strings.TrimSpace(reference)
		if reference == "" {
			return nil, fmt.Errorf("catalog source is empty")
		}

		source := reference
		if !IsOCIReference(reference) {
			resolved, err := utils.GetFullPath(reference, options.CWD)
			if err != nil {
				return nil, fmt.Errorf("get catalog path: %w", err)
			}
			source = resolved
		}
		if _, exists := seen[source]; exists {
			continue
		}
		seen[source] = struct{}{}
		sources = append(sources, source)
	}

	return sources, nil
}
