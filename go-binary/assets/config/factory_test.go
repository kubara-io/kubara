package config

import (
	"kubara/assets/envmap"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClusterFromEnv(t *testing.T) {
	// --- Test Data Setup ---
	// 1. Create a sample environment map that will be the input to the function.
	sampleEnvMap := &envmap.EnvMap{
		ProjectName:           "kubara-test",
		ProjectStage:          "dev",
		DomainName:            "example.com",
		StackitProjectId:      "123e4567-e89b-12d3-a456-426614174000",
		PrivateLoadbalancerIp: "192.168.1.10",
		PublicLoadbalancerIp:  "203.0.113.10",
		ArgocdGitHttpsUrl:     "https://github.com/org/repo.git",
	}

	// 2. Manually construct the expected Cluster struct based on the sampleEnvMap.
	// This is what we expect the function to return.
	expectedDNSName := "kubara-test-dev.example.com"
	expectedCluster := Cluster{
		Name:                  "kubara-test",
		Stage:                 "dev",
		ProjectID:             "123e4567-e89b-12d3-a456-426614174000",
		Type:                  "<controlplane or workerplane>",
		DNSName:               expectedDNSName,
		PrivateLoadBalancerIP: "192.168.1.10",
		PublicLoadBalancerIP:  "203.0.113.10",
		SSOOrg:                "<my-org>",
		SSOTeam:               "<my-team>",
		Terraform: &Terraform{
			KubernetesType:    "<edge or ske>",
			KubernetesVersion: "1.34",
			DNS: DNS{
				Name:  expectedDNSName,
				Email: "my-test@nowhere.com",
			},
		},
		ArgoCD: ArgoCD{Repo: RepoProto{
			HTTPS: &RepoType{
				Customer: Repository{
					URL:            "https://github.com/org/repo.git",
					TargetRevision: "main",
				},
				Managed: Repository{
					URL:            "https://github.com/org/repo.git",
					TargetRevision: "main",
				},
			},
		}},
		// The statuses of services are hardcoded in the function, so we mirror them here.
		Services: Services{
			Argocd:              GenericService{ServiceStatus{Status: StatusDisabled}},
			CertManager:         CertManagerService{ServiceStatus: ServiceStatus{Status: StatusEnabled}, ClusterIssuer: ClusterIssuer{Name: "letsencrypt-staging", Email: "yourname@your-domain.de", Server: "https://acme-staging-v02.api.letsencrypt.org/directory"}},
			ExternalDns:         GenericService{ServiceStatus: ServiceStatus{Status: StatusEnabled}},
			ExternalSecrets:     GenericService{ServiceStatus: ServiceStatus{Status: StatusEnabled}},
			KubePrometheusStack: GenericService{ServiceStatus: ServiceStatus{Status: StatusEnabled}},
			IngressNginx:        GenericService{ServiceStatus: ServiceStatus{Status: StatusEnabled}},
			Kyverno:             GenericService{ServiceStatus: ServiceStatus{Status: StatusEnabled}},
			KyvernoPolicies:     GenericService{ServiceStatus: ServiceStatus{Status: StatusEnabled}},
			KyvernoPolicyReport: GenericService{ServiceStatus: ServiceStatus{Status: StatusEnabled}},
			Loki:                GenericService{ServiceStatus: ServiceStatus{Status: StatusEnabled}},
			HomerDashboard:      GenericService{ServiceStatus: ServiceStatus{Status: StatusEnabled}},
			Oauth2Proxy:         GenericService{ServiceStatus: ServiceStatus{Status: StatusEnabled}},
			MetricsServer:       GenericService{ServiceStatus: ServiceStatus{Status: StatusDisabled}},
			MetalLb:             GenericService{ServiceStatus: ServiceStatus{Status: StatusDisabled}},
			Longhorn:            GenericService{ServiceStatus: ServiceStatus{Status: StatusDisabled}},
		},
	}

	// --- Test Cases Definition ---
	type args struct {
		e *envmap.EnvMap
	}
	tests := []struct {
		name string
		args args
		want Cluster
	}{
		{
			name: "should correctly create a cluster config from a given EnvMap",
			args: args{
				e: sampleEnvMap,
			},
			want: expectedCluster,
		},
	}

	// --- Test Execution ---
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewClusterFromEnv(tt.args.e), "NewClusterFromEnv(%v) should return the expected Cluster struct", tt.args.e)
		})
	}
}
