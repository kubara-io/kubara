package cluster

import "github.com/urfave/cli/v3"

func NewClusterCommand() *cli.Command {
	return &cli.Command{
		Name:        "cluster",
		Usage:       "Manage clusters for Kubara",
		UsageText:   "kubara cluster [command]",
		Description: "Enables handling for hub and spoke clusters",
		Commands: []*cli.Command{
			CreateClusterList(),
			AddCluster(),
		},
	}
}
