package cluster

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	internal "github.com/kubara-io/kubara/internal/cmd/cluster"
	"github.com/kubara-io/kubara/internal/utils"
	"github.com/urfave/cli/v3"
)

// Creates the Command for the 'kubara cluster add command
// Command necessitates a cluster-name as an arg
func AddCluster() *cli.Command {
	return &cli.Command{
		Name:        "add",
		Usage:       "Create a new spoke cluster for the hub",
		UsageText:   "kubara cluster add <cluster-name>",
		Description: "Adds a new spoke cluster to the existing hub cluster",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "cluster-name",
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
		},

		Action: func(c context.Context, cmd *cli.Command) error {
			spokeName := cmd.StringArg("cluster-name")
			if len(spokeName) == 0 {
				cli.ShowSubcommandHelpAndExit(cmd, 1)
			}

			cwd, err := filepath.Abs(cmd.String("work-dir"))
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			rawCatalog := strings.TrimSpace(cmd.String("catalog"))
			catalogOptions := catalog.LoadOptions{Overwrite: cmd.Bool("catalog-overwrite")}
			if rawCatalog != "" {
				catalogOptions.CatalogPath, err = utils.GetFullPath(rawCatalog, cwd)
				if err != nil {
					return fmt.Errorf("get catalog path: %w", err)
				}
			}

			configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
			if err != nil {
				return fmt.Errorf("get config file path: %w", err)
			}

			return internal.AddCluster(configFilePath, spokeName, catalogOptions)
		},
	}
}
