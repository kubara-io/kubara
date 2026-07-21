package workflow

import (
	"fmt"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
)

func CreateOrUpdateClusterFromEnvWithCatalog(cfg *config.Config, e *envconfig.EnvMap, catalogOptions catalog.LoadOptions) error {
	clusterName := e.ProjectName

	// Attempt to find the cluster to update
	for i := range cfg.Clusters {
		if cfg.Clusters[i].Name == clusterName {
			fmt.Printf("Found existing cluster '%s', updating fields...\n", clusterName)

			// Apply the new values from the environment to the found cluster.
			gitRepoURL := e.GitRepositoryURL()
			cfg.Clusters[i].Stage = e.ProjectStage
			cfg.Clusters[i].ArgoCD.Repo.AuthMode = e.GitAuthMode()
			cfg.Clusters[i].ArgoCD.Repo.Git.Configs.URL = gitRepoURL
			cfg.Clusters[i].ArgoCD.Repo.Git.Components.URL = gitRepoURL

			if envconfig.IsConfiguredEnvValue(e.ArgocdHelmRepoUrl) {
				helmRepoURL := envconfig.NormalizeHelmRepoURL(e.ArgocdHelmRepoUrl)
				cfg.Clusters[i].ArgoCD.HelmRepo = &config.HelmRepository{
					URL: helmRepoURL,
				}
			}

			return nil
		}
	}

	// If the loop completes without returning, the cluster was not found.
	fmt.Printf("No cluster named '%s' found, creating a new one...\n", clusterName)
	newCluster, err := config.NewClusterFromEnvWithCatalog(e, catalogOptions)
	if err != nil {
		return err
	}
	cfg.Clusters = append(cfg.Clusters, newCluster)
	return nil
}
