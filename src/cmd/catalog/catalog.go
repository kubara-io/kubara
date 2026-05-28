package catalog

import (
	"github.com/urfave/cli/v3"
)

func NewCatalogCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "catalog",
		Usage:       "Manage custom catalogs and service definitions",
		UsageText:   "kubara catalog [command]",
		Description: "Provides commands to scaffold custom catalogs and add service definition manifests within them.",
		Commands: []*cli.Command{
			NewCatalogCreate(),
			NewCatalogService(),
			//NewCatalogList(),
			//NewCatalogPull(),
			//NewCatalogPush(),
		},
	}

	return cmd
}
