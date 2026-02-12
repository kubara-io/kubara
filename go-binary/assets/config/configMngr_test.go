package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewConfigManager(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     *Manager
	}{
		{
			name:     "Create a new config manager",
			filePath: "/tmp/config.yaml",
			want: &Manager{
				filepath: "/tmp/config.yaml",
				config:   &Config{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewConfigManager(tt.filePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestManager_Load(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// 1. Valid YAML content for the success case
	validYAML := `
clusters:
  - name: test-cluster
    stage: dev
	ingressClassName: traefik
    projectId: "123e4567-e89b-12d3-a456-426614174000"
    type: controlplane
    dnsName: test-cluster.example.com
    ingressClassName: traefik
    argocd:
      repo:
        https:
          customer:
            url: "https://github.com/customer/repo.git"
            targetRevision: "main"
          managed:
            url: "https://github.com/managed/repo.git"
            targetRevision: "main"
    services:
      certManager:
        status: "enabled"
        clusterIssuer:
          name: "letsencrypt-prod"
          email: "admin@example.com"
          server: "https://acme-v02.api.letsencrypt.org/directory"
`
	validFilepath := filepath.Join(tempDir, "valid_config.yaml")
	// Use require.NoError for setup, as the test cannot proceed if this fails.
	require.NoError(t, os.WriteFile(validFilepath, []byte(validYAML), 0644), "Failed to create valid config file")

	// Expected struct for the successful load case
	expectedConfig := &Config{
		Clusters: []Cluster{
			{
				Name:             "test-cluster",
				Stage:            "dev",
				IngressClassName: "traefik",
				ProjectID:        "123e4567-e89b-12d3-a456-426614174000",
				Type:             "controlplane",
				DNSName:          "test-cluster.example.com",
				ArgoCD: ArgoCD{
					Repo: RepoProto{
						HTTPS: &RepoType{
							Customer: Repository{
								URL:            "https://github.com/customer/repo.git",
								TargetRevision: "main",
							},
							Managed: Repository{
								URL:            "https://github.com/managed/repo.git",
								TargetRevision: "main",
							},
						},
					},
				},
				Services: Services{
					CertManager: CertManagerService{
						ServiceStatus: ServiceStatus{Status: StatusEnabled},
						ClusterIssuer: ClusterIssuer{
							Name:   "letsencrypt-prod",
							Email:  "admin@example.com",
							Server: "https://acme-v02.api.letsencrypt.org/directory",
						},
					},
				},
			},
		},
	}

	// 2. Malformed YAML for the invalid format case
	invalidYAML := `clusters: [name: invalid`
	invalidYAMLFilepath := filepath.Join(tempDir, "invalid_yaml.yaml")
	require.NoError(t, os.WriteFile(invalidYAMLFilepath, []byte(invalidYAML), 0644), "Failed to create invalid yaml file")

	// 3. Valid YAML with a data type mismatch
	mismatchYAML := `
clusters:
  - name: 12345 # This should be a string
    stage: dev
	ingressClassName: traefik
    projectId: "123e4567-e89b-12d3-a456-426614174000"
    type: controlplane
    dnsName: test-cluster.example.com
    argocd: {}
    services: {}
`
	mismatchFilepath := filepath.Join(tempDir, "mismatch.yaml")
	require.NoError(t, os.WriteFile(mismatchFilepath, []byte(mismatchYAML), 0644), "Failed to create mismatch config file")

	// --- Test Cases Definition ---
	tests := []struct {
		name       string
		filepath   string
		wantConfig *Config
		wantErr    bool
	}{
		{
			name:       "Success: Correctly loads a valid config file",
			filepath:   validFilepath,
			wantConfig: expectedConfig,
			wantErr:    false,
		},
		{
			name:     "Error: File does not exist",
			filepath: filepath.Join(tempDir, "non_existent_file.yaml"),
			wantErr:  true,
		},
		{
			name:     "Error: File has invalid YAML format",
			filepath: invalidYAMLFilepath,
			wantErr:  true,
		},
		{
			name:     "Error: File has data type mismatch",
			filepath: mismatchFilepath,
			wantErr:  true,
		},
	}

	// --- Test Execution ---
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewConfigManager(tt.filepath)
			err := cm.Load()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantConfig, cm.GetConfig())
			}
		})
	}
}

func TestManager_Validate(t *testing.T) {
	// A base valid config to be used in tests.
	validConfig := &Config{
		Clusters: []Cluster{
			{
				Name:             "test-cluster",
				Stage:            "dev",
				IngressClassName: "traefik",
				ProjectID:        "123e4567-e89b-12d3-a456-426614174000",
				Type:             "controlplane",
				DNSName:          "test-cluster.example.com",
				Terraform: &Terraform{
					KubernetesType:    "ske",
					KubernetesVersion: "1.34",
					DNS: DNS{
						Name:  "example.com",
						Email: "admin@example.com",
					},
				},
				ArgoCD: ArgoCD{
					Repo: RepoProto{
						HTTPS: &RepoType{
							Customer: Repository{
								URL:            "https://github.com/customer/repo.git",
								TargetRevision: "main",
							},
							Managed: Repository{
								URL:            "https://github.com/managed/repo.git",
								TargetRevision: "main",
							},
						},
					},
				},
				Services: Services{
					CertManager: CertManagerService{
						ServiceStatus: ServiceStatus{Status: StatusEnabled},
						ClusterIssuer: ClusterIssuer{
							Name:   "letsencrypt-prod",
							Email:  "cert@example.com",
							Server: "https://acme-v02.api.letsencrypt.org/directory",
						},
					},
					Argocd:              GenericService{ServiceStatus{Status: StatusEnabled}},
					ExternalDns:         GenericService{ServiceStatus{Status: StatusEnabled}},
					ExternalSecrets:     GenericService{ServiceStatus{Status: StatusEnabled}},
					KubePrometheusStack: GenericService{ServiceStatus{Status: StatusEnabled}},
					Traefik:             GenericService{ServiceStatus{Status: StatusEnabled}},
					Kyverno:             GenericService{ServiceStatus{Status: StatusEnabled}},
					KyvernoPolicies:     GenericService{ServiceStatus{Status: StatusEnabled}},
					KyvernoPolicyReport: GenericService{ServiceStatus{Status: StatusEnabled}},
					Loki:                GenericService{ServiceStatus{Status: StatusEnabled}},
					HomerDashboard:      GenericService{ServiceStatus{Status: StatusEnabled}},
					Oauth2Proxy:         GenericService{ServiceStatus{Status: StatusEnabled}},
					MetricsServer:       GenericService{ServiceStatus{Status: StatusEnabled}},
					MetalLb:             GenericService{ServiceStatus{Status: StatusEnabled}},
					Longhorn:            GenericService{ServiceStatus{Status: StatusEnabled}},
				},
			},
		},
	}

	// Helper function to deep copy the valid config
	deepCopy := func(c *Config) *Config {
		newConfig := *c
		newConfig.Clusters = make([]Cluster, len(c.Clusters))
		copy(newConfig.Clusters, c.Clusters)
		return &newConfig
	}

	// --- Test Cases ---
	invalidConfigMissingField := deepCopy(validConfig)
	invalidConfigMissingField.Clusters[0].Name = "" // Name is required

	invalidConfigPatternMismatch := deepCopy(validConfig)
	invalidConfigPatternMismatch.Clusters[0].ProjectID = "not-a-valid-uuid"

	invalidConfigEnumMismatch := deepCopy(validConfig)
	invalidConfigEnumMismatch.Clusters[0].Type = "invalid-type"

	invalidConfigFormatMismatch := deepCopy(validConfig)
	clonedTerraform := *invalidConfigFormatMismatch.Clusters[0].Terraform
	clonedTerraform.DNS.Email = "not-an-email"
	invalidConfigFormatMismatch.Clusters[0].Terraform = &clonedTerraform

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid_config_should_pass_validation",
			config:  validConfig,
			wantErr: false,
		},
		{
			name:    "invalid_config_should_fail_on_missing_required_field",
			config:  invalidConfigMissingField,
			wantErr: true,
		},
		{
			name:    "invalid_config_should_fail_on_pattern_mismatch",
			config:  invalidConfigPatternMismatch,
			wantErr: true,
		},
		{
			name:    "invalid_config_should_fail_on_enum_mismatch",
			config:  invalidConfigEnumMismatch,
			wantErr: true,
		},
		{
			name:    "invalid_config_should_fail_on_format_mismatch",
			config:  invalidConfigFormatMismatch,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &Manager{
				config: tt.config,
			}
			err := cm.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_SaveToFile(t *testing.T) {
	// --- Test Data Setup ---
	testConfig := &Config{
		Clusters: []Cluster{
			{
				Name:             "prod-cluster",
				Stage:            "production",
				IngressClassName: "traefik",
				ProjectID:        "123e4567-e89b-12d3-a456-426614174000",
				Type:             "controlplane",
				DNSName:          "prod.example.com",
				// Keeping other fields zero for a clean test case
				ArgoCD:   ArgoCD{},
				Services: Services{},
			},
		},
	}

	// --- Test Environment Setup ---
	// Create a temporary directory for all test files. It will be cleaned up automatically.
	tempDir := t.TempDir()

	// Filepath for the successful save case
	successfulFilepath := filepath.Join(tempDir, "config.yaml")

	// Setup for the permission error case: a read-only directory
	readOnlyDir := filepath.Join(tempDir, "readonly_dir")
	// Use require here because the test cannot proceed if setup fails
	require.NoError(t, os.Mkdir(readOnlyDir, 0755))
	require.NoError(t, os.Chmod(readOnlyDir, 0555)) // r-x r-x r-x
	permissionErrorFilepath := filepath.Join(readOnlyDir, "config.yaml")

	// --- Test Cases Definition ---
	type fields struct {
		filepath string
		config   *Config
	}
	tests := []struct {
		name      string
		fields    fields
		wantErr   assert.ErrorAssertionFunc
		postCheck func(t *testing.T, filepath string) // Add a function for checks after the save attempt
	}{
		{
			name: "Success: Correctly saves a valid config to a new file",
			fields: fields{
				filepath: successfulFilepath,
				config:   testConfig,
			},
			wantErr: assert.NoError,
			postCheck: func(t *testing.T, filepath string) {
				// Verify that the file was actually created.
				assert.FileExists(t, filepath)

				// Read the file and unmarshal it to verify its content.
				savedBytes, err := os.ReadFile(filepath)
				require.NoError(t, err, "Failed to read the newly saved file")

				var savedConfig Config
				err = yaml.Unmarshal(savedBytes, &savedConfig)
				require.NoError(t, err, "Saved file content should be valid YAML")

				// Assert that the content saved to the file matches the original config.
				assert.Equal(t, testConfig, &savedConfig)
			},
		},
		{
			name: "Error: Fails when trying to save to a read-only directory",
			fields: fields{
				filepath: permissionErrorFilepath,
				config:   testConfig,
			},
			wantErr: assert.Error,
			postCheck: func(t *testing.T, filepath string) {
				// Verify that the file was not created.
				assert.NoFileExists(t, filepath)
			},
		},
	}

	// --- Test Execution ---
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &Manager{
				filepath: tt.fields.filepath,
				config:   tt.fields.config,
			}

			// This executes the assertion (assert.NoError or assert.Error) on the result of cm.SaveToFile()
			err := cm.SaveToFile()
			tt.wantErr(t, err, fmt.Sprintf("SaveToFile() with filepath %s", tt.fields.filepath))

			// If a post-check function is defined for the test case, run it.
			if tt.postCheck != nil {
				tt.postCheck(t, tt.fields.filepath)
			}
		})
	}
}

func TestManager_GetFilepath(t *testing.T) {
	type fields struct {
		filepath string
		config   *Config
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Success: Correctly gets the filepath",
			fields: fields{
				filepath: "some-file.yaml",
				config:   &Config{},
			},
			want: "some-file.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &Manager{
				filepath: tt.fields.filepath,
				config:   tt.fields.config,
			}
			assert.Equalf(t, tt.want, cm.GetFilepath(), "GetFilepath()")
		})
	}
}
