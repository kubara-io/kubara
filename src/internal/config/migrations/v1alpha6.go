package migrations

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

// migrateV1Alpha6Config migrates v1alpha6 configurations to the v1alpha7 schema.
// It moves clusters[].argocd from the hub cluster to a top-level gitops configuration,
// setting gitops.engine = "argocd", copying selfManaged, repo, and helmRepo,
// and removing the argocd key from all clusters.
func migrateV1Alpha6Config(config map[string]any) error {
	log.Info().Msg("migrating config from v1alpha6 format to v1alpha7")

	clusters, _ := config["clusters"].([]any)

	// If gitops is not already set, find argocd from the hub cluster.
	if _, hasGitOps := config["gitops"]; !hasGitOps {
		var hubArgoCD map[string]any
		for _, item := range clusters {
			cluster, ok := item.(map[string]any)
			if !ok {
				continue
			}
			clusterType, _ := cluster["type"].(string)
			if clusterType == "hub" || clusterType == "" {
				if argoCD, ok := cluster["argocd"].(map[string]any); ok {
					hubArgoCD = argoCD
					break
				}
			}
		}

		if hubArgoCD == nil {
			return fmt.Errorf("no argocd configuration found on hub cluster to migrate to top-level gitops")
		}

		gitops := map[string]any{
			"engine": "argocd",
		}
		if selfManaged, ok := hubArgoCD["selfManaged"]; ok {
			gitops["selfManaged"] = selfManaged
		}
		if repo, ok := hubArgoCD["repo"]; ok {
			gitops["repo"] = repo
		}
		if helmRepo, ok := hubArgoCD["helmRepo"]; ok {
			gitops["helmRepo"] = helmRepo
		}
		if clusterSecrets, ok := hubArgoCD["clusterSecrets"]; ok {
			gitops["clusterSecrets"] = clusterSecrets
		}
		config["gitops"] = gitops
	}

	// Remove argocd key from all clusters
	for _, item := range clusters {
		cluster, ok := item.(map[string]any)
		if !ok {
			continue
		}
		delete(cluster, "argocd")
	}

	config["version"] = ConfigVersionV1Alpha7
	return nil
}
