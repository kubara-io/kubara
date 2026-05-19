package catalog

import (
	"github.com/urfave/cli/v3"
)

func NewCatalogCommand() *cli.Command {
	cmd := &cli.Command{
		Name:        "catalog",
		Usage:       "",
		UsageText:   "",
		Description: "",
		Commands: []*cli.Command{
			NewCatalogCreate(),
			//NewCatalogList(),
			//NewCatalogPull(),
			//NewCatalogPush(),
		},
	}

	return cmd
}
