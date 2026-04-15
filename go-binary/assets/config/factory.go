package config

import (
	"kubara/assets/catalog"
	"kubara/assets/envmap"
)

// NewClusterFromEnv creates a new Cluster configuration populated with default
// values and information from an EnvMap.
func NewClusterFromEnv(e *envmap.EnvMap) (Cluster, error) {
	return NewClusterFromEnvWithCatalog(e, catalog.LoadOptions{})
}

func NewClusterFromEnvWithCatalog(e *envmap.EnvMap, catalogOptions catalog.LoadOptions) (Cluster, error) {
	dnsName := e.ProjectName + "-" + e.ProjectStage + "." + e.DomainName
	services, err := newDefaultServicesFromCatalogWithOptions(catalogOptions, "")
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
