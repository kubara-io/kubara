package cluster

import "github.com/urfave/cli/v3"

func NewClusterCommand() *cli.Command {
	return &cli.Command{
		Name:        "cluster",
		Usage:       "Manage your kubara cluster configurations",
		UsageText:   "kubara cluster [command]",
		Description: "Enables the configuration and quick setup of clusters",
		Commands: []*cli.Command{
			CreateClusterList(),
			AddCluster(),
		},
	}
}
