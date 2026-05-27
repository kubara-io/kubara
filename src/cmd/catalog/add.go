package catalog

import (
	"context"
	"github.com/urfave/cli/v3"

	internal "github.com/kubara-io/kubara/internal/cmd/catalog"
)

func NewCatalogService() *cli.Command {
	cmd := &cli.Command{
		Name:        "add",
		Usage:       "Add a service definition to the current catalog",
		UsageText:   "kubara catalog add SERVICE_NAME",
		Description: "Creates services/SERVICE_NAME.yaml in the current catalog. Run this command from a catalog root that already contains Catalog.yaml.",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "service-name",
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
		},
		Action: func(c context.Context, cmd *cli.Command) error {
			serviceName := cmd.StringArg("service-name")
			if len(serviceName) == 0 {
				cli.ShowSubcommandHelpAndExit(cmd, 1)
			}

			return internal.CreateService(serviceName)
		},
	}

	return cmd
}
