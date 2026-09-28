package migrations

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// migrateV1Alpha4Config migrates configurations with version ConfigVersionV1Alpha4 to the ConfigVersionV1Alpha5 schema format.
// It renames argocd.repo.https to argocd.repo.git.
func migrateV1Alpha4Config(config map[string]any) error {
	log.Info().Msg("migrating config from v1alpha4 format to v1alpha5")
	config["version"] = ConfigVersionV1Alpha5

	clusters, ok := config["clusters"].([]any)
	if !ok {
		return nil
	}

	for i, clusterRaw := range clusters {
		cluster, ok := clusterRaw.(map[string]any)
		if !ok {
			continue
		}
		if err := migrateRepoHTTPSKey(cluster, i); err != nil {
			return fmt.Errorf("cannot migrate cluster number %d: %w", i, err)
		}
	}

	return nil
}

func migrateRepoHTTPSKey(cluster map[string]any, clusterIndex int) error {
	argocdRaw, ok := cluster["argocd"]
	if !ok || argocdRaw == nil {
		return nil
	}
	argocd, ok := argocdRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.argocd must be an object", clusterLabel(cluster, clusterIndex))
	}

	repoRaw, ok := argocd["repo"]
	if !ok || repoRaw == nil {
		return nil
	}
	repo, ok := repoRaw.(map[string]any)
	if !ok {
		return fmt.Errorf("%s.argocd.repo must be an object", clusterLabel(cluster, clusterIndex))
	}

	httpsRepo, hasHTTPS := repo["https"]
	if !hasHTTPS {
		return nil
	}
	if _, hasGit := repo["git"]; hasGit {
		return fmt.Errorf("%s.argocd.repo has both legacy https and git repositories", clusterLabel(cluster, clusterIndex))
	}

	repo["git"] = httpsRepo
	delete(repo, "https")
	return nil
}
