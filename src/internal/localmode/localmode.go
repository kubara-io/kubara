package localmode

import (
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/service"
)

const (
	DomainName          = "traefik.me"
	DefaultProjectStage = "local"
	ExampleGitRepoURL   = "https://github.com/your-user/your-gitops-repo.git"
	DefaultSSOOrg       = "local"
	DefaultSSOTeam      = "local"
)

var (
	enabledServices = map[string]struct{}{
		"cert-manager":          {},
		"external-secrets":      {},
		"homer-dashboard":       {},
		"kube-prometheus-stack": {},
		"metrics-server":        {},
		"traefik":               {},
	}
)

func DefaultDNSName(projectName, projectStage string) string {
	return projectName + "-" + projectStage + "." + DomainName
}

func ApplyClusterProfile(cluster *config.Cluster, dnsName string) {
	cluster.Type = "hub"
	cluster.DNSName = dnsName
	cluster.SSOOrg = DefaultSSOOrg
	cluster.SSOTeam = DefaultSSOTeam
	cluster.IngressClassName = "traefik" //nolint:staticcheck // Keep the deprecated field in sync for existing catalogs during the transition.
	cluster.Networking = config.ClusterNetworking{
		Type: config.NetworkingIngress,
		Ingress: &config.IngressNetworking{
			ClassName: "traefik",
		},
	}
	cluster.Terraform = nil

	for serviceName, serviceConfig := range cluster.Services {
		serviceConfig.Status = service.StatusDisabled
		if serviceConfig.Networking != nil {
			networking := *serviceConfig.Networking
			networking.Gateway = nil
			serviceConfig.Networking = &networking
		}
		cluster.Services[serviceName] = serviceConfig
	}
	for serviceName := range enabledServices {
		if serviceConfig, exists := cluster.Services[serviceName]; exists {
			serviceConfig.Status = service.StatusEnabled
			cluster.Services[serviceName] = serviceConfig
		}
	}
}
