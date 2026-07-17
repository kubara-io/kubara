package cluster

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/utils"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func CreateAddClusterCommand() *cli.Command {
	return &cli.Command{
		Name:        "add",
		Usage:       "Add a new spoke cluster to your config",
		UsageText:   "kubara cluster add CLUSTER_NAME",
		Description: "Adds a new spoke cluster to an existing config yaml",
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

			catalogOptions, err := catalog.ResolveLoadOptions(cwd, cmd.StringSlice("catalog"), cmd.Bool("catalog-overwrite"))
			if err != nil {
				return fmt.Errorf("could not resolve catalog options: %w", err)
			}

			configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
			if err != nil {
				return fmt.Errorf("get config file path: %w", err)
			}

			configStore := config.NewConfigStore(cwd, configFilePath, catalogOptions)
			err = configStore.Load()
			if err != nil {
				return fmt.Errorf("config load: %w", err)
			}
			currentConfig := configStore.GetConfig()

			clusters := currentConfig.Clusters
			for _, existing := range clusters {
				if existing.Name == spokeName {
					return fmt.Errorf("cluster %q already exists", spokeName)
				}
			}

			newCluster := config.CreateSpokeScaffolding(spokeName)

			currentConfig.Clusters = append(clusters, newCluster)

			if err = configStore.ApplyServiceCatalogDefaults(); err != nil {
				return fmt.Errorf("apply spoke catalog defaults: %w", err)
			}
			if err = configStore.SaveToFile(); err != nil {
				return fmt.Errorf("save config to file: %w", err)
			}

			log.Info().Msgf("Spoke cluster %q has been successfully added to the config", spokeName)
			return nil
		},
	}
}
