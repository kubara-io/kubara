package catalog

import (
	"context"
	"github.com/urfave/cli/v3"

	internal "github.com/kubara-io/kubara/internal/cmd/catalog"
)

func NewCatalogCreate() *cli.Command {
	cmd := &cli.Command{
		Name:        "create",
		Usage:       "Create a custom catalog directory skeleton",
		UsageText:   "kubara catalog create CATALOG_NAME",
		Description: "Scaffolds a custom catalog directory with Catalog.yaml plus customer-service-catalog, managed-service-catalog, and services directories.",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "catalog-name",
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
		},
		Action: func(c context.Context, cmd *cli.Command) error {
			catalogName := cmd.StringArg("catalog-name")
			if len(catalogName) == 0 {
				cli.ShowSubcommandHelpAndExit(cmd, 1)
			}

			return internal.CreateCatalog(catalogName)
		},
	}

	return cmd
}
