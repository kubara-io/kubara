package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/urfave/cli/v3"
)

func catalogLoadOptionsFromCommand(cmd *cli.Command) (catalog.LoadOptions, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return catalog.LoadOptions{}, fmt.Errorf("failed to get working directory: %w", err)
	}

	rawCatalogPath := strings.TrimSpace(cmd.String("catalog"))
	if rawCatalogPath == "" {
		return catalog.LoadOptions{
			CatalogPath: "",
			Overwrite:   cmd.Bool("catalog-overwrite"),
		}, nil
	}

	absoluteCatalogPath, err := utils.GetFullPath(rawCatalogPath, cwd)
	if err != nil {
		return catalog.LoadOptions{}, fmt.Errorf("failed to get catalog path: %w", err)
	}

	return catalog.LoadOptions{
		CatalogPath: absoluteCatalogPath,
		Overwrite:   cmd.Bool("catalog-overwrite"),
	}, nil
}
