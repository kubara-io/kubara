package cluster

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"

	"github.com/kubara-io/kubara/internal/utils"
	"github.com/urfave/cli/v3"
)

func CreateClusterList() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List all clusters in the config file",
		UsageText:   "kubara cluster ls",
		Description: "List all clusters available in the current config.yaml file",
		Aliases:     []string{"ls"},
		Action: func(c context.Context, cmd *cli.Command) error {
			ws, err := utils.ResolveWorkspaceFromCommand(cmd)
			if err != nil {
				return fmt.Errorf("resolve workspace: %w", err)
			}

			catalogOptions, err := catalog.ResolveLoadOptions(ws.WorkDir, "", cmd.StringSlice("catalog"), cmd.Bool("catalog-overwrite"))
			if err != nil {
				return fmt.Errorf("could not resolve catalog options: %w", err)
			}

			configStore := config.NewConfigStore(ws.WorkDir, ws.ConfigFilePath, catalogOptions)
			err = configStore.Load()
			if err != nil {
				return fmt.Errorf("config load: %w", err)
			}

			clusters := configStore.GetConfig().Clusters

			writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, err = fmt.Fprintln(writer, "NAME\tTYPE\tPROVIDER")
			if err != nil {
				return fmt.Errorf("print table head into buffer: %w", err)
			}
			for _, cluster := range clusters {
				provider := config.TerraformProviderNone
				if cluster.Terraform != nil {
					provider = cluster.Terraform.Provider
				}
				_, err = fmt.Fprintf(
					writer,
					"%s\t%s\t%s\n",
					cluster.Name,
					cluster.Type,
					string(provider),
				)
				if err != nil {
					return fmt.Errorf("print list into buffer: %w", err)
				}
			}
			return writer.Flush()
		},
	}
}
