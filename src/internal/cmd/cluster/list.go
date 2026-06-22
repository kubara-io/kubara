package cluster

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
)

// ListClusters is an internal Function for the 'kubara cluster ls' command
func ListClusters(configFilePath string) error {
	configStore := config.NewConfigStoreWithCatalog(configFilePath, catalog.LoadOptions{})
	err := configStore.Load()
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
}
