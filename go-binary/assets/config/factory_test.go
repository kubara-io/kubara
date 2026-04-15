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
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		DomainName:        "example.com",
		ArgocdGitHttpsUrl: "https://github.com/org/repo.git",
		ArgocdHelmRepoUrl: "https://charts.example.com",
	}
	sampleEnvMapWithoutHelmRepo := &envmap.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		DomainName:        "example.com",
		ArgocdGitHttpsUrl: "https://github.com/org/repo.git",
	}
	sampleEnvMapWithOCIHelmRepo := &envmap.EnvMap{
		ProjectName:       "kubara-test",
		ProjectStage:      "dev",
		DomainName:        "example.com",
		ArgocdGitHttpsUrl: "https://github.com/org/repo.git",
		ArgocdHelmRepoUrl: "oci://registry-1.docker.io/bitnamicharts",
	}

	// 2. Manually construct the expected Cluster struct based on the sampleEnvMap.
	// This is what we expect the function to return.
	expectedDNSName := "kubara-test-dev.example.com"
	expectedCluster := Cluster{
		Name:             "kubara-test",
		Stage:            "dev",
		Type:             "<controlplane or worker>",
		DNSName:          expectedDNSName,
		SSOOrg:           "<my-org>",
		SSOTeam:          "<my-team>",
		IngressClassName: "traefik",
		Terraform: &Terraform{
			Provider:          "<provider>",
			ProjectID:         "<project-id>",
			KubernetesType:    "<edge or ske>",
			KubernetesVersion: "1.34",
			DNS: DNS{
				Name:  expectedDNSName,
				Email: "my-test@nowhere.com",
			},
		},
		ArgoCD: ArgoCD{
			Repo: RepoProto{
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
			},
			HelmRepo: &HelmRepository{
				URL: "https://charts.example.com",
			},
		},
		// The service defaults are catalog-driven; mirror expected built-in values.
		Services: Services{
			"argo-cd":                 {Status: StatusDisabled},
			"cert-manager":            {Status: StatusEnabled, Config: map[string]any{"clusterIssuer": map[string]any{"name": "letsencrypt-staging", "email": "yourname@your-domain.de", "server": "https://acme-staging-v02.api.letsencrypt.org/directory"}}},
			"external-dns":            {Status: StatusEnabled},
			"external-secrets":        {Status: StatusEnabled},
			"kube-prometheus-stack":   {Status: StatusEnabled},
			"traefik":                 {Status: StatusEnabled},
			"kyverno":                 {Status: StatusEnabled},
			"kyverno-policies":        {Status: StatusEnabled},
			"kyverno-policy-reporter": {Status: StatusEnabled},
			"loki":                    {Status: StatusEnabled},
			"homer-dashboard":         {Status: StatusEnabled},
			"oauth2-proxy":            {Status: StatusEnabled},
			"metrics-server":          {Status: StatusDisabled},
			"metallb":                 {Status: StatusDisabled},
			"longhorn":                {Status: StatusDisabled},
		},
	}
	expectedClusterWithoutHelmRepo := expectedCluster
	expectedClusterWithoutHelmRepo.ArgoCD.HelmRepo = nil
	expectedClusterWithOCIHelmRepo := expectedCluster
	expectedClusterWithOCIHelmRepo.ArgoCD.HelmRepo = &HelmRepository{
		URL: "registry-1.docker.io/bitnamicharts",
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
		{
			name: "should not set helmRepo when no helm repo URL is provided",
			args: args{
				e: sampleEnvMapWithoutHelmRepo,
			},
			want: expectedClusterWithoutHelmRepo,
		},
		{
			name: "should normalize oci helm repo URL",
			args: args{
				e: sampleEnvMapWithOCIHelmRepo,
			},
			want: expectedClusterWithOCIHelmRepo,
		},
	}

	// --- Test Execution ---
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NewClusterFromEnv(tt.args.e), "NewClusterFromEnv(%v) should return the expected Cluster struct", tt.args.e)
		})
	}
}
