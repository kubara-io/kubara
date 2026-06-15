package cluster

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
)

// Internal Function for the 'kubara cluster ls' command
// Requires the configFilePath for the ConfigStore
// Prints out the context in tabular form
func ListClusters(configFilePath string) error {

	configStore := config.NewConfigStoreWithCatalog(configFilePath, catalog.LoadOptions{})
	err := configStore.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	clusters := configStore.GetConfig().Clusters

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tTYPE\tPROVIDER")
	for _, cluster := range clusters {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\n",
			cluster.Name,
			cluster.Type,
			cluster.Terraform.Provider,
		)
	}
	return writer.Flush()
}
