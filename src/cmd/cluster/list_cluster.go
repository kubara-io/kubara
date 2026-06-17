package cluster

import (
	"context"
	"fmt"
	"path/filepath"

	internal "github.com/kubara-io/kubara/internal/cmd/cluster"
	"github.com/kubara-io/kubara/internal/utils"
	"github.com/urfave/cli/v3"
)

func CreateClusterList() *cli.Command {
	return &cli.Command{
		Name:        "ls",
		Usage:       "List all the cluster in the config file",
		UsageText:   "kubara cluster ls",
		Description: "List all the clusters available in the current config.yaml file",
		Action: func(c context.Context, cmd *cli.Command) error {
			//TODO: add command line flag for config yaml
			cwd, err := filepath.Abs(cmd.String("work-dir"))
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
			if err != nil {
				return fmt.Errorf("get config file path: %w", err)
			}

			return internal.ListClusters(configFilePath)
		},
	}
}
