package cluster

import (
	"fmt"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
)

// AddCluster is an internal Function for the 'kubara cluster add [spoke-name]' command
func AddCluster(configFilePath string, spokeName string, catalogOptions catalog.LoadOptions) error {
	configStore := config.NewConfigStoreWithCatalog(configFilePath, catalogOptions)
	err := configStore.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}
	currentConfig := configStore.GetConfig()

	clusters := currentConfig.Clusters

	newCluster := config.CreateBlankSpokeCluster(spokeName)

	currentConfig.Clusters = append(clusters, newCluster)

	if err = configStore.ApplyServiceCatalogDefaults(); err != nil {
		return fmt.Errorf("apply spoke catalog defaults: %w", err)
	}
	if err = configStore.SaveToFile(); err != nil {
		return fmt.Errorf("save config to file: %w", err)
	}

	return nil
}
