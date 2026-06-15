package cluster

import (
	"fmt"
	"maps"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/service"
)

func AddCluster(configFilePath string, spokeName string) error {

	configStore := config.NewConfigStoreWithCatalog(configFilePath, catalog.LoadOptions{})
	err := configStore.Load()
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}
	currentConfig := configStore.GetConfig()

	clusters := currentConfig.Clusters
	hubCluster := findHubCluster(clusters)

	//create new cluster and append it
	newCluster := hubCluster
	newCluster.Name = spokeName
	newCluster.Type = "spoke"

	spokeServices := maps.Clone(newCluster.Services)
	newCluster.Services = disableServicesFor(spokeServices, []string{"homer-dashboard", "argocd"})

	currentConfig.Clusters = append(clusters, newCluster)

	configStore.SaveToFile()

	return nil
}

// findHubCluster looks for the Hub cluster in a list of clusters and returns it
func findHubCluster(clusters []config.Cluster) config.Cluster {
	for _, cluster := range clusters {
		if cluster.Type == "hub" {
			return cluster
		}
	}
	// base case, only necessary for compiler,
	// should not execute in a default environment
	return config.Cluster{}
}

// disableServicesFor, receives a service map and a list of serviceNames
// Goes through the list of service names and for each sets the status to disabled
// Returns an updated list of serviceName
func disableServicesFor(services map[string]service.Service, serviceNames []string) service.Services {
	for _, name := range serviceNames {
		svc := services[name]
		svc.Status = service.StatusDisabled
		services[name] = svc
	}
	return services
}
