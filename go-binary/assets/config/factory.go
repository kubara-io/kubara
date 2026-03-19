package config

import "kubara/assets/envmap"

// NewClusterFromEnv creates a new Cluster configuration populated with default
// values and information from an EnvMap.
func NewClusterFromEnv(e *envmap.EnvMap) Cluster {
	dnsName := e.ProjectName + "-" + e.ProjectStage + "." + e.DomainName
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
		argoCD.HelmRepo = &HelmRepository{
			URL: e.ArgocdHelmRepoUrl,
		}
	}

	return Cluster{
		Name:             e.ProjectName,
		Stage:            e.ProjectStage,
		Type:             "<controlplane or workerplane>",
		DNSName:          dnsName,
		SSOOrg:           "<my-org>",
		SSOTeam:          "<my-team>",
		IngressClassName: "traefik",
		Terraform: &Terraform{
			ProjectID:         "<project-id>",
			KubernetesType:    "<edge or ske>",
			KubernetesVersion: "1.34",
			DNS: DNS{
				Name:  dnsName,
				Email: "my-test@nowhere.com",
			},
		},
		ArgoCD: argoCD,
		Services: Services{
			Argocd: GenericService{ServiceStatus{Status: StatusDisabled}},
			CertManager: CertManagerService{
				ServiceStatus: ServiceStatus{
					Status: StatusEnabled,
				},
				ClusterIssuer: ClusterIssuer{
					Name:   "letsencrypt-staging",
					Email:  "yourname@your-domain.de",
					Server: "https://acme-staging-v02.api.letsencrypt.org/directory",
				},
			},
			ExternalDns: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusEnabled,
				},
			},
			ExternalSecrets: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusEnabled,
				},
			},
			KubePrometheusStack: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusEnabled,
				},
			},
			Traefik: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusEnabled,
				},
			},
			Kyverno: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusEnabled,
				},
			},
			KyvernoPolicies: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusEnabled,
				},
			},
			KyvernoPolicyReport: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusEnabled,
				},
			},
			Loki: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusEnabled,
				},
			},
			HomerDashboard: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusEnabled,
				},
			},
			Oauth2Proxy: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusEnabled,
				},
			},
			MetricsServer: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusDisabled,
				},
			},
			MetalLb: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusDisabled,
				},
			},
			Longhorn: GenericService{
				ServiceStatus: ServiceStatus{
					Status: StatusDisabled,
				},
			},
		},
	}
}
