package config

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/service"
	"github.com/kubara-io/kubara/internal/utils"
)

// InferGitRemote attempts to detect the git remote.origin.url and infer its GitAuthMode.
func InferGitRemote(cwd string) (string, GitAuthMode) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", cwd, "config", "--get", "remote.origin.url")
	out, err := cmd.Output()
	if err == nil {
		rawURL := strings.TrimSpace(string(out))
		if rawURL != "" {
			if utils.IsSSHGitURL(rawURL) {
				return rawURL, GitAuthModeSSH
			}
			return rawURL, GitAuthModeHTTPS
		}
	}

	return "", GitAuthModeHTTPS
}

// NewDefaultClusterWithCatalog creates a new Cluster scaffold for kubara init.
func NewDefaultClusterWithCatalog(cwd string, catalogOptions catalog.LoadOptions) (Cluster, error) {
	effectiveCatalogOptions := clusterCatalogLoadOptions(catalogOptions)
	services, err := createServicesFromCatalogWithOptions(effectiveCatalogOptions, "")
	if err != nil {
		return Cluster{}, fmt.Errorf("create services from catalog: %w", err)
	}

	inferredURL, authMode := InferGitRemote(cwd)
	gitRepoURL := inferredURL
	if gitRepoURL == "" {
		gitRepoURL = "https://git.example.com/platform/repo.git"
	}

	return Cluster{
		Name:    "<cluster-name>",
		Stage:   "<stage>",
		Type:    Hub,
		DNSName: "<subdomain.my-domain.com>",
		SSOOrg:  "<my-org>",
		SSOTeam: "<my-team>",
		Networking: ClusterNetworking{
			Type: NetworkingIngress,
			Ingress: &IngressNetworking{
				ClassName: "traefik",
			},
		},
		Terraform: &Terraform{
			Provider:          TerraformProviderNone,
			ProjectID:         "<project-id>",
			KubernetesType:    "<edge, ske or cce>",
			KubernetesVersion: "1.36",
			DNS: DNS{
				Name:  "<subdomain.my-domain.com>",
				Email: "[EMAIL_REDACTED]",
			},
		},
		ArgoCD: ArgoCD{
			SelfManaged: ArgoCDSelfManagedEnabled,
			Repo: RepoProto{
				AuthMode: authMode,
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
		},
		Catalogs: append([]string(nil), effectiveCatalogOptions.Catalogs...),
		Services: services,
	}, nil
}

func clusterCatalogLoadOptions(catalogOptions catalog.LoadOptions) catalog.LoadOptions {
	effective := catalogOptions
	if len(effective.Catalogs) == 0 {
		effective.Catalogs = []string{catalog.DefaultGeneralCatalog}
		return effective
	}

	effective.Catalogs = append([]string(nil), effective.Catalogs...)
	return effective
}

func createServicesFromCatalogWithOptions(catalogOptions catalog.LoadOptions, clusterType string) (service.Services, error) {
	cat, err := catalog.Load(catalogOptions)
	if err != nil {
		return nil, fmt.Errorf("load catalog: %w", err)
	}
	cat = cat.UserConfigurableServices()

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

func CreateSpokeScaffolding(name string, catalogOptions catalog.LoadOptions) Cluster {
	effectiveCatalogOptions := clusterCatalogLoadOptions(catalogOptions)

	return Cluster{
		Name:    name,
		Stage:   "<stage>",
		Type:    "spoke",
		DNSName: "<dns-name>",
		SSOOrg:  "<my-org>",
		SSOTeam: "<my-team>",
		Networking: ClusterNetworking{
			Type: NetworkingIngress,
			Ingress: &IngressNetworking{
				ClassName: "traefik",
			},
		},
		Terraform: &Terraform{
			Provider:          "<provider>",
			ProjectID:         "<project-id>",
			KubernetesType:    "<edge or ske>",
			KubernetesVersion: "<version>",
			DNS: DNS{
				Name:  "<dns-name>",
				Email: "<dns-mail>",
			},
		},
		ArgoCD: ArgoCD{
			SelfManaged: ArgoCDSelfManagedEnabled,
			Repo: RepoProto{
				AuthMode: GitAuthModeHTTPS,
				Git: &RepoType{
					Configs:    Repository{URL: "https://git.example.com/platform/repo.git", TargetRevision: "main"},
					Components: Repository{URL: "https://git.example.com/platform/repo.git", TargetRevision: "main"},
				},
			},
		},
		Catalogs: append([]string(nil), effectiveCatalogOptions.Catalogs...),
		Services: nil,
	}
}
