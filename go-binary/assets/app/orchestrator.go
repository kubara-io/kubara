package app

import (
	"fmt"
	"kubara/assets/config"
	"kubara/assets/envmap"
)

// CreateOrUpdateClusterFromEnv finds a cluster by name and updates it,
// or creates it if it doesn't exist.
func CreateOrUpdateClusterFromEnv(cfg *config.Config, e *envmap.EnvMap) {
	clusterName := e.ProjectName
	dnsName := e.ProjectName + "-" + e.ProjectStage + "." + e.DomainName

	// Attempt to find the cluster to update
	for i := range cfg.Clusters {
		if cfg.Clusters[i].Name == clusterName {
			fmt.Printf("Found existing cluster '%s', updating fields...\n", clusterName)

			// Apply the new values from the environment to the found cluster.
			cfg.Clusters[i].Stage = e.ProjectStage
			cfg.Clusters[i].ProjectID = e.StackitProjectId
			cfg.Clusters[i].DNSName = dnsName
			cfg.Clusters[i].PrivateLoadBalancerIP = e.PrivateLoadbalancerIp
			cfg.Clusters[i].PublicLoadBalancerIP = e.PublicLoadbalancerIp
			cfg.Clusters[i].Terraform.DNS.Name = dnsName
			cfg.Clusters[i].ArgoCD.Repo.HTTPS.Managed.URL = e.ArgocdGitHttpsUrl
			cfg.Clusters[i].ArgoCD.Repo.HTTPS.Customer.URL = e.ArgocdGitHttpsUrl

			return
		}
	}

	// If the loop completes without returning, the cluster was not found.
	fmt.Printf("No cluster named '%s' found, creating a new one...\n", clusterName)
	newCluster := config.NewClusterFromEnv(e)
	cfg.Clusters = append(cfg.Clusters, newCluster)
}
