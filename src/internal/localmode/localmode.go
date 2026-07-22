package localmode

import (
	"strings"

	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/service"
)

const (
	DomainName            = "traefik.me"
	DefaultProjectStage   = "local"
	ExampleGitRepoURL     = "https://github.com/your-user/your-gitops-repo.git"
	ExampleWizardPassword = "CHANGE_ME"
	DefaultSSOOrg         = "local"
	DefaultSSOTeam        = "local"
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

func PopulateInitEnv(env *envconfig.EnvMap) {
	if !envconfig.IsConfiguredEnvValue(env.ProjectName) {
		env.ProjectName = "test-cluster"
	}
	if !envconfig.IsConfiguredEnvValue(env.ProjectStage) {
		env.ProjectStage = DefaultProjectStage
	}
	if !envconfig.IsConfiguredEnvValue(env.ArgocdGitHttpsUrl) {
		env.ArgocdGitHttpsUrl = ExampleGitRepoURL
	}
	if !envconfig.IsConfiguredEnvValue(env.ArgocdWizardAccountPassword) {
		env.ArgocdWizardAccountPassword = ExampleWizardPassword
	}
}

func IsGeneratedPlaceholder(v string) bool {
	trimmed := strings.TrimSpace(v)
	return trimmed == ExampleGitRepoURL ||
		trimmed == ExampleWizardPassword
}

func ApplyClusterProfile(cluster *config.Cluster, dnsName string) {
	cluster.Type = "hub"
	cluster.DNSName = dnsName
	cluster.SSOOrg = DefaultSSOOrg
	cluster.SSOTeam = DefaultSSOTeam
	cluster.IngressClassName = "traefik"
	cluster.Terraform = nil

	for serviceName, serviceConfig := range cluster.Services {
		serviceConfig.Status = service.StatusDisabled
		cluster.Services[serviceName] = serviceConfig
	}
	for serviceName := range enabledServices {
		if serviceConfig, exists := cluster.Services[serviceName]; exists {
			serviceConfig.Status = service.StatusEnabled
			cluster.Services[serviceName] = serviceConfig
		}
	}
}
