package config

import (
	"fmt"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/service"
)

func NewClusterFromEnvWithCatalog(e *envconfig.EnvMap, catalogOptions catalog.LoadOptions) (Cluster, error) {
	gitRepoURL := e.GitRepositoryURL()
	services, err := createServicesFromCatalogWithOptions(catalogOptions, "")
	if err != nil {
		return Cluster{}, fmt.Errorf("create services from catalog: %w", err)
	}

	argoCD := ArgoCD{
		Repo: RepoProto{
			AuthMode: e.GitAuthMode(),
			Git: &RepoType{
				Configs: Repository{
					URL:            gitRepoURL,
					TargetRevision: "main",
				},
				Components: Repository{
					URL:            gitRepoURL,
					TargetRevision: "main",
				},
			},
		},
	}
	if envconfig.IsConfiguredEnvValue(e.ArgocdHelmRepoUrl) {
		helmRepoURL := envconfig.NormalizeHelmRepoURL(e.ArgocdHelmRepoUrl)
		argoCD.HelmRepo = &HelmRepository{
			URL: helmRepoURL,
		}
	}

	return Cluster{
		Name:             e.ProjectName,
		Stage:            e.ProjectStage,
		Type:             "<hub or spoke>",
		DNSName:          "<subdomain.my-domain.com>",
		SSOOrg:           "<my-org>",
		SSOTeam:          "<my-team>",
		IngressClassName: "traefik",
		Terraform: &Terraform{
			Provider:          TerraformProviderNone,
			ProjectID:         "<project-id>",
			KubernetesType:    "<edge, ske or cce>",
			KubernetesVersion: "1.34",
			DNSContactEmail:   "my-test@nowhere.com",
		},
		ArgoCD:   argoCD,
		Services: services,
	}, nil
}

func createServicesFromCatalogWithOptions(catalogOptions catalog.LoadOptions, clusterType string) (service.Services, error) {
	cat, err := catalog.Load(catalogOptions)
	if err != nil {
		return nil, fmt.Errorf("load catalog: %w", err)
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
			return nil, fmt.Errorf("apply defaults for service %q: %w", name, err)
		}

		services[name] = service.Service{
			Status: def.Spec.Status,
			Config: cfg,
		}
	}

	return services, nil
}

func CreateSpokeScaffolding(name string) Cluster {
	return Cluster{
		Name:    name,
		Stage:   "<stage>",
		Type:    "spoke",
		DNSName: "<dns-name>",
		SSOOrg:  "<my-org>",
		SSOTeam: "<my-team>",
		Terraform: &Terraform{
			Provider:          "<provider>",
			ProjectID:         "<project-id>",
			KubernetesType:    "<edge or ske>",
			KubernetesVersion: "<version>",
			DNSContactEmail:   "<dns-mail>",
		},
		ArgoCD: ArgoCD{
			Repo: RepoProto{
				AuthMode: envconfig.GitAuthModeHTTPS,
				Git: &RepoType{
					Configs:    Repository{URL: "https://git.example.com/platform/repo.git", TargetRevision: "main"},
					Components: Repository{URL: "https://git.example.com/platform/repo.git", TargetRevision: "main"},
				},
			},
		},
		Services: nil,
	}
}
