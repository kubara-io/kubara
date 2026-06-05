package catalog

import (
	"github.com/urfave/cli/v3"
)

func NewCatalogCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "catalog",
		Usage:       "Manage custom catalogs and service definitions",
		UsageText:   "kubara catalog [command]",
		Description: "Provides commands to scaffold, package, pull, push, list, and unpackage catalogs as local directories or OCI artifacts.",
		Commands: []*cli.Command{
			NewCatalogCreate(),
			NewCatalogService(),
			//NewCatalogPull(),
			//NewCatalogPush(),
			NewCatalogList(),
			NewCatalogPackage(),
			NewCatalogUnpackage(),
		},
	}

	return cmd
}
