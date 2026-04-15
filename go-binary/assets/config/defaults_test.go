package config

import (
	"testing"

	"kubara/assets/catalog"

	"github.com/stretchr/testify/assert"
)

func TestApplyDefaults_ClusterLevelDefaults(t *testing.T) {
	cfg := &Config{
		Clusters: []Cluster{
			{
				Name:    "test",
				DNSName: "test.example.com",
				// Should get defaults for:
				// Stage, Type, IngressClassName
			},
		},
	}

	applyDefaults(cfg)

	c := cfg.Clusters[0]
	assert.Equal(t, "dev", c.Stage, "Stage should default to dev")
	assert.Equal(t, "controlplane", c.Type, "Type should default to controlplane")
	assert.Equal(t, "traefik", c.IngressClassName, "IngressClassName should default to traefik")
}

func TestApplyDefaults_DoesNotOverwriteExplicitValues(t *testing.T) {
	cfg := &Config{
		Clusters: []Cluster{
			{
				Name:             "test",
				Stage:            "production",
				Type:             "worker",
				IngressClassName: "nginx",
				DNSName:          "test.example.com",
			},
		},
	}

	applyDefaults(cfg)

	c := cfg.Clusters[0]
	assert.Equal(t, "production", c.Stage)
	assert.Equal(t, "worker", c.Type)
	assert.Equal(t, "nginx", c.IngressClassName)
}

func TestApplyDefaults_NestedTerraformDefaults(t *testing.T) {
	cfg := &Config{
		Clusters: []Cluster{
			{
				Terraform: &Terraform{
					ProjectID:         "some-id",
					KubernetesVersion: "1.34",
					DNS:               DNS{Name: "example.com", Email: "admin@example.com"},
					// Should get defaults for:
					// Provider and KubernetesType
				},
			},
		},
	}

	applyDefaults(cfg)

	tf := cfg.Clusters[0].Terraform
	assert.Equal(t, "stackit", tf.Provider, "Provider should default to stackit")
	assert.Equal(t, "ske", tf.KubernetesType, "KubernetesType should default to ske")
}

func TestApplyDefaults_NilPointerStaysNil(t *testing.T) {
	cfg := &Config{
		Clusters: []Cluster{
			{
				Terraform: nil, // should remain nil, not be allocated
			},
		},
	}

	applyDefaults(cfg)

	assert.Nil(t, cfg.Clusters[0].Terraform, "nil Terraform pointer should stay nil")
}

func TestApplyDefaults_RepositoryTargetRevision(t *testing.T) {
	cfg := &Config{
		Clusters: []Cluster{
			{
				ArgoCD: ArgoCD{
					Repo: RepoProto{
						HTTPS: &RepoType{
							Customer: Repository{URL: "https://github.com/customer/repo.git"},
							Managed:  Repository{URL: "https://github.com/managed/repo.git", TargetRevision: "release"},
						},
					},
				},
			},
		},
	}

	applyDefaults(cfg)

	https := cfg.Clusters[0].ArgoCD.Repo.HTTPS
	assert.Equal(t, "main", https.Customer.TargetRevision, "empty TargetRevision should default to main")
	assert.Equal(t, "release", https.Managed.TargetRevision, "explicit TargetRevision should not be overwritten")
}

func TestApplyDefaults_EmbeddedServiceStatusDefaults(t *testing.T) {
	cfg := &Config{
		Clusters: []Cluster{
			{
				Services: Services{
					"argo-cd":      {},
					"cert-manager": {},
				},
			},
		},
	}

	applyDefaults(cfg)
	err := applyServiceCatalogDefaults(cfg, catalog.LoadOptions{})
	assert.NoError(t, err)

	assert.Equal(t, StatusDisabled, cfg.Clusters[0].Services["argo-cd"].Status, "empty service status should default to disabled")
	assert.Equal(t, StatusEnabled, cfg.Clusters[0].Services["cert-manager"].Status, "empty cert-manager status should default to built-in default")
}

func TestApplyDefaults_WorkerClusterServicesDefaultToDisabled(t *testing.T) {
	cfg := &Config{
		Clusters: []Cluster{
			{
				Type:     "worker",
				Services: Services{},
			},
		},
	}

	applyDefaults(cfg)
	err := applyServiceCatalogDefaults(cfg, catalog.LoadOptions{})
	assert.NoError(t, err)

	assert.Equal(t, StatusDisabled, cfg.Clusters[0].Services["cert-manager"].Status)
	assert.Equal(t, StatusDisabled, cfg.Clusters[0].Services["external-dns"].Status)
	assert.Equal(t, StatusDisabled, cfg.Clusters[0].Services["traefik"].Status)
}

func TestApplyDefaults_WorkerClusterKeepsExplicitServiceStatus(t *testing.T) {
	cfg := &Config{
		Clusters: []Cluster{
			{
				Type: "worker",
				Services: Services{
					"cert-manager": {Status: StatusEnabled},
				},
			},
		},
	}

	applyDefaults(cfg)
	err := applyServiceCatalogDefaults(cfg, catalog.LoadOptions{})
	assert.NoError(t, err)

	assert.Equal(t, StatusEnabled, cfg.Clusters[0].Services["cert-manager"].Status)
}

func TestApplyDefaults_ClusterIssuerDefaults(t *testing.T) {
	cfg := &Config{
		Clusters: []Cluster{
			{
				Services: Services{
					"cert-manager": {
						Config: map[string]any{
							"clusterIssuer": map[string]any{
								"email": "cert@example.com",
							},
						},
					},
				},
			},
		},
	}

	applyDefaults(cfg)
	err := applyServiceCatalogDefaults(cfg, catalog.LoadOptions{})
	assert.NoError(t, err)

	issuer, _ := cfg.Clusters[0].Services["cert-manager"].Config["clusterIssuer"].(map[string]any)
	assert.Equal(t, "letsencrypt-staging", issuer["name"], "ClusterIssuer Name should default to letsencrypt-staging")
	assert.Equal(t, "https://acme-staging-v02.api.letsencrypt.org/directory", issuer["server"], "ClusterIssuer Server should default to ACME staging URL")
}

func TestApplyDefaults_MultipleSliceElements(t *testing.T) {
	cfg := &Config{
		Clusters: []Cluster{
			{Name: "cluster-a"},
			{Name: "cluster-b", Stage: "prod"},
		},
	}

	applyDefaults(cfg)

	assert.Equal(t, "dev", cfg.Clusters[0].Stage, "first cluster Stage should be defaulted")
	assert.Equal(t, "prod", cfg.Clusters[1].Stage, "second cluster Stage should keep explicit value")
	assert.Equal(t, "controlplane", cfg.Clusters[0].Type)
	assert.Equal(t, "controlplane", cfg.Clusters[1].Type)
}

func TestParseDefaultFromTag(t *testing.T) {
	tests := []struct {
		tag    string
		want   string
		wantOK bool
	}{
		{"default=dev", "dev", true},
		{"required,default=traefik,minLength=1", "traefik", true},
		{"title=ACME Server,format=uri,default=https://acme-staging-v02.api.letsencrypt.org/directory", "https://acme-staging-v02.api.letsencrypt.org/directory", true},
		{"required,minLength=1", "", false},
		{"", "", false},
		{"enum=enabled,enum=disabled,default=disabled", "disabled", true},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got, ok := parseDefaultFromTag(tt.tag)
			assert.Equal(t, tt.wantOK, ok)
			if ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
