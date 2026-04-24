package config

import (
	"fmt"

	"kubara/internal/assets/envmap"
	"kubara/internal/catalog"
	"kubara/internal/assets/service"
)

// NewClusterFromEnv creates a new Cluster configuration populated with default
// values and information from an EnvMap.
func NewClusterFromEnv(e *envmap.EnvMap) (Cluster, error) {
	return NewClusterFromEnvWithCatalog(e, catalog.LoadOptions{})
}

func NewClusterFromEnvWithCatalog(e *envmap.EnvMap, catalogOptions catalog.LoadOptions) (Cluster, error) {
	dnsName := e.ProjectName + "-" + e.ProjectStage + "." + e.DomainName
	services, err := createServicesFromCatalogWithOptions(catalogOptions, "")
	if err != nil {
		return Cluster{}, err
	}

	argoCD := ArgoCD{
		Repo: RepoProto{
			HTTPS: &RepoType{
				Customer: Repository{
					URL:            e.ArgocdGitHttpsUrl,
					TargetRevision: "main",
				},
				Managed: Repository{
					URL:            e.ArgocdGitHttpsUrl,
					TargetRevision: "main",
				},
			},
		},
	}
	if envmap.IsConfiguredEnvValue(e.ArgocdHelmRepoUrl) {
		helmRepoURL := envmap.NormalizeHelmRepoURL(e.ArgocdHelmRepoUrl)
		argoCD.HelmRepo = &HelmRepository{
			URL: helmRepoURL,
		}
	}

	return Cluster{
		Name:             e.ProjectName,
		Stage:            e.ProjectStage,
		Type:             "<controlplane or worker>",
		DNSName:          dnsName,
		SSOOrg:           "<my-org>",
		SSOTeam:          "<my-team>",
		IngressClassName: "traefik",
		Terraform: &Terraform{
			Provider:          "<provider>",
			ProjectID:         "<project-id>",
			KubernetesType:    "<edge or ske>",
			KubernetesVersion: "1.34",
			DNS: DNS{
				Name:  dnsName,
				Email: "my-test@nowhere.com",
			},
		},
		ArgoCD:   argoCD,
		Services: services,
	}, nil
}

func createServicesFromCatalogWithOptions(catalogOptions catalog.LoadOptions, clusterType string) (service.Services, error) {
	cat, err := catalog.Load(catalogOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to load catalog: %w", err)
	}

	services := make(service.Services, len(cat.Services))
	for name, def := range cat.Services {
		if clusterType != "" && len(def.Spec.ClusterTypes) > 0 {
			include := false
			for _, allowed := range def.Spec.ClusterTypes {
				if allowed == clusterType {
					include = true
					break
				}
			}
			if !include {
				continue
			}
		}

		cfg, err := applySchemaDefaults(def.Spec.ConfigSchema, map[string]any{})
		if err != nil {
			return nil, fmt.Errorf("failed to apply defaults for service %q: %w", name, err)
		}

		services[name] = service.Service{
			Status: def.Spec.Status,
			Config: cfg,
		}
	}

	return services, nil
}
