package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubara-io/kubara/internal/service"

	schemaValidator "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// Helper function to create a valid test config
func newValidTestConfig() *Config {
	return &Config{
		Version: ConfigVersionV1Alpha1,
		Clusters: []Cluster{
			{
				Name:             "test-cluster",
				Stage:            "dev",
				IngressClassName: "traefik",
				Type:             "hub",
				DNSName:          "test-cluster.example.com",
				Terraform: &Terraform{
					Provider:          "stackit",
					ProjectID:         "00000000-0000-0000-0000-000000000000",
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
				Services: service.Services{
					"argocd":                  {Status: service.StatusEnabled},
					"cert-manager":            {Status: service.StatusEnabled, Config: service.Config{"clusterIssuer": map[string]any{"name": "letsencrypt-prod", "email": "cert@example.com", "server": "https://acme-v02.api.letsencrypt.org/directory"}}},
					"external-dns":            {Status: service.StatusEnabled},
					"external-secrets":        {Status: service.StatusEnabled},
					"kube-prometheus-stack":   {Status: service.StatusEnabled, Storage: &service.Storage{ClassName: "standard-rwo"}},
					"traefik":                 {Status: service.StatusEnabled},
					"kyverno":                 {Status: service.StatusEnabled},
					"kyverno-policies":        {Status: service.StatusEnabled},
					"kyverno-policy-reporter": {Status: service.StatusEnabled},
					"loki":                    {Status: service.StatusEnabled, Storage: &service.Storage{ClassName: "standard-rwo"}},
					"homer-dashboard":         {Status: service.StatusEnabled},
					"oauth2-proxy":            {Status: service.StatusEnabled},
					"metrics-server":          {Status: service.StatusEnabled},
					"metallb":                 {Status: service.StatusEnabled},
					"longhorn":                {Status: service.StatusEnabled},
				},
			},
		},
	}
}

// Helper function to deep copy a config
func deepCopyConfig(c *Config) *Config {
	newConfig := *c
	newConfig.Clusters = make([]Cluster, len(c.Clusters))
	copy(newConfig.Clusters, c.Clusters)
	return &newConfig
}

func TestNewConfigStore(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     *ConfigStore
	}{
		{
			name:     "Create a new config store",
			filePath: "/tmp/config.yaml",
			want: &ConfigStore{
				filepath: "/tmp/config.yaml",
				config:   &Config{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewConfigStore(tt.filePath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigStore_Load(t *testing.T) {
	tempDir := t.TempDir()

	expectedConfig := newValidTestConfig()

	validYAML, err := yaml.Marshal(expectedConfig)
	require.NoError(t, err, "Failed to marshal valid config to YAML")

	validFilepath := filepath.Join(tempDir, "valid_config.yaml")
	require.NoError(t, os.WriteFile(validFilepath, validYAML, 0644), "Failed to create valid config file")

	// Malformed YAML syntax
	invalidYAML := `clusters: [name: invalid`
	invalidYAMLFilepath := filepath.Join(tempDir, "invalid_yaml.yaml")
	require.NoError(t, os.WriteFile(invalidYAMLFilepath, []byte(invalidYAML), 0644), "Failed to create invalid yaml file")

	// Valid YAML but wrong data types (name should be string, not int)
	mismatchYAML := `
clusters:
  - name: 12345
    stage: dev
    type: hub
    dnsName: test-cluster.example.com
    ingressClassName: traefik
    terraform:
      projectId: "00000000-0000-0000-0000-000000000000"
    argocd: {}
    services: {}
`
	mismatchFilepath := filepath.Join(tempDir, "mismatch.yaml")
	require.NoError(t, os.WriteFile(mismatchFilepath, []byte(mismatchYAML), 0644), "Failed to create mismatch config file")

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := NewConfigStore(tt.filepath)
			err := cs.Load()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantConfig, cs.GetConfig())
			}
		})
	}
}

func TestConfigStore_LoadMigratesLegacyConfig(t *testing.T) {
	legacyYAML := `
clusters:
  - name: legacy-cluster
    dnsName: legacy.example.com
    type: hub
    argocd:
      repo:
        https:
          customer:
            url: "https://github.com/customer/repo.git"
          managed:
            url: "https://github.com/managed/repo.git"
    services:
      argocd:
        status: enabled
      certManager:
        status: enabled
        clusterIssuer:
          name: letsencrypt-staging
          email: cert@example.com
          server: https://acme-staging-v02.api.letsencrypt.org/directory
      kubePrometheusStack:
        status: enabled
        storageClassName: metrics-rwo
      loki:
        status: enabled
        storageClassName: logs-rwo
      oauth2Proxy:
        status: enabled
        ingress:
          annotations:
            foo: bar
`

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "legacy-config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(legacyYAML), 0644))

	cs := NewConfigStore(configPath)
	require.NoError(t, cs.Load())

	loaded := cs.GetConfig()
	require.Equal(t, ConfigVersionV1Alpha1, loaded.Version)
	require.Len(t, loaded.Clusters, 1)

	cluster := loaded.Clusters[0]
	assert.Equal(t, cluster.Type, "hub")
	assert.Contains(t, cluster.Services, "argocd")

	certManager := cluster.Services["cert-manager"]
	clusterIssuer, ok := certManager.Config["clusterIssuer"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "letsencrypt-staging", clusterIssuer["name"])
	assert.Equal(t, "cert@example.com", clusterIssuer["email"])

	require.NotNil(t, cluster.Services["kube-prometheus-stack"].Storage)
	assert.Equal(t, "metrics-rwo", cluster.Services["kube-prometheus-stack"].Storage.ClassName)

	require.NotNil(t, cluster.Services["loki"].Storage)
	assert.Equal(t, "logs-rwo", cluster.Services["loki"].Storage.ClassName)

	require.NotNil(t, cluster.Services["oauth2-proxy"].Networking)
	assert.Equal(t, "bar", cluster.Services["oauth2-proxy"].Networking.Annotations["foo"])

	savedBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)
	savedContent := string(savedBytes)
	assert.Contains(t, savedContent, "version: v1alpha1")
	assert.Contains(t, savedContent, "cert-manager:")
	assert.Contains(t, savedContent, "argocd:")
	assert.NotContains(t, savedContent, "certManager:")
	assert.NotContains(t, savedContent, "storageClassName:")
	assert.NotContains(t, savedContent, "ingress:")
	assert.Contains(t, savedContent, "className: metrics-rwo")
	assert.Contains(t, savedContent, "className: logs-rwo")
	assert.Contains(t, savedContent, "networking:")
	assert.Contains(t, savedContent, "clusterIssuer:")
}

func TestConfigStore_LoadRejectsLegacyMigrationConflicts(t *testing.T) {
	tests := []struct {
		name        string
		servicesYML string
		wantErr     string
	}{
		{
			name: "duplicate canonical service names",
			servicesYML: `
      certManager:
        status: enabled
      cert-manager:
        status: enabled
`,
			wantErr: `conflicting keys "certManager" and "cert-manager"`,
		},
		{
			name: "cert-manager clusterIssuer conflict",
			servicesYML: `
      certManager:
        status: enabled
        clusterIssuer:
          name: letsencrypt-staging
        config:
          clusterIssuer:
            name: letsencrypt-prod
`,
			wantErr: "both legacy clusterIssuer and config.clusterIssuer",
		},
		{
			name: "storage class conflict",
			servicesYML: `
      loki:
        status: enabled
        storageClassName: logs-rwo
        storage:
          className: already-set
`,
			wantErr: "both legacy storageClassName and storage.className",
		},
		{
			name: "ingress annotations conflict",
			servicesYML: `
      oauth2Proxy:
        status: enabled
        ingress:
          annotations:
            foo: bar
        networking:
          annotations:
            custom: value
`,
			wantErr: "both legacy ingress.annotations and networking.annotations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			legacyYAML := fmt.Sprintf(`
clusters:
  - name: legacy-cluster
    dnsName: legacy.example.com
    argocd:
      repo:
        https:
          customer:
            url: "https://github.com/customer/repo.git"
          managed:
            url: "https://github.com/managed/repo.git"
    services:%s`, tt.servicesYML)

			configPath := filepath.Join(t.TempDir(), "legacy-conflict.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(legacyYAML), 0644))

			cs := NewConfigStore(configPath)
			err := cs.Load()
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestConfigStore_Validate(t *testing.T) {
	validConfig := newValidTestConfig()

	// Test required field validation
	invalidConfigMissingField := deepCopyConfig(validConfig)
	invalidConfigMissingField.Clusters[0].Name = ""

	// Test pattern validation (version format)
	invalidConfigPatternMismatch := deepCopyConfig(validConfig)
	clonedTerraformPattern := *invalidConfigPatternMismatch.Clusters[0].Terraform
	clonedTerraformPattern.KubernetesVersion = "not-a-valid-version"
	invalidConfigPatternMismatch.Clusters[0].Terraform = &clonedTerraformPattern

	// Test enum validation
	invalidConfigEnumMismatch := deepCopyConfig(validConfig)
	invalidConfigEnumMismatch.Clusters[0].Type = "invalid-type"

	// Test format validation (email)
	invalidConfigFormatMismatch := deepCopyConfig(validConfig)
	clonedTerraform := *invalidConfigFormatMismatch.Clusters[0].Terraform
	clonedTerraform.DNS.Email = "not-an-email"
	invalidConfigFormatMismatch.Clusters[0].Terraform = &clonedTerraform

	// Terraform is optional at the cluster level
	validConfigWithoutTerraform := deepCopyConfig(validConfig)
	validConfigWithoutTerraform.Clusters[0].Terraform = nil

	// But if Terraform is present, its required fields must be set
	invalidConfigMissingTerraformField := deepCopyConfig(validConfig)
	clonedTerraformMissing := *invalidConfigMissingTerraformField.Clusters[0].Terraform
	clonedTerraformMissing.ProjectID = ""
	invalidConfigMissingTerraformField.Clusters[0].Terraform = &clonedTerraformMissing

	// Test optional IP address fields
	validConfigWithLoadBalancerIPs := deepCopyConfig(validConfig)
	validConfigWithLoadBalancerIPs.Clusters[0].PrivateLoadBalancerIP = "192.168.1.10"
	validConfigWithLoadBalancerIPs.Clusters[0].PublicLoadBalancerIP = "203.0.113.10"

	invalidConfigInvalidPrivateIP := deepCopyConfig(validConfig)
	invalidConfigInvalidPrivateIP.Clusters[0].PrivateLoadBalancerIP = "not-an-ip"

	invalidConfigInvalidPublicIP := deepCopyConfig(validConfig)
	invalidConfigInvalidPublicIP.Clusters[0].PublicLoadBalancerIP = "999.999.999.999"

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
			name:    "valid_config_without_terraform_should_pass_validation",
			config:  validConfigWithoutTerraform,
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
		{
			name:    "invalid_config_should_fail_on_missing_terraform_required_field",
			config:  invalidConfigMissingTerraformField,
			wantErr: true,
		},
		{
			name:    "valid_config_with_loadbalancer_ips_should_pass_validation",
			config:  validConfigWithLoadBalancerIPs,
			wantErr: false,
		},
		{
			name:    "invalid_config_should_fail_on_invalid_private_loadbalancer_ip",
			config:  invalidConfigInvalidPrivateIP,
			wantErr: true,
		},
		{
			name:    "invalid_config_should_fail_on_invalid_public_loadbalancer_ip",
			config:  invalidConfigInvalidPublicIP,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ConfigStore{
				config: tt.config,
			}
			err := cs.validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigStore_SaveToFile(t *testing.T) {
	testConfig := &Config{
		Clusters: []Cluster{
			{
				Name:             "prod-cluster",
				Stage:            "production",
				IngressClassName: "traefik",
				Type:             "hub",
				DNSName:          "prod.example.com",
				Terraform: &Terraform{
					ProjectID: "00000000-0000-0000-0000-000000000000",
				},
				ArgoCD:   ArgoCD{},
				Services: service.Services{},
			},
		},
	}

	tempDir := t.TempDir()

	successfulFilepath := filepath.Join(tempDir, "config.yaml")

	// Create a read-only directory to test permission errors
	readOnlyDir := filepath.Join(tempDir, "readonly_dir")
	require.NoError(t, os.Mkdir(readOnlyDir, 0755))
	require.NoError(t, os.Chmod(readOnlyDir, 0555))
	permissionErrorFilepath := filepath.Join(readOnlyDir, "config.yaml")

	type fields struct {
		filepath string
		config   *Config
	}
	tests := []struct {
		name      string
		fields    fields
		wantErr   assert.ErrorAssertionFunc
		postCheck func(t *testing.T, filepath string)
	}{
		{
			name: "Success: Correctly saves a valid config to a new file",
			fields: fields{
				filepath: successfulFilepath,
				config:   testConfig,
			},
			wantErr: assert.NoError,
			postCheck: func(t *testing.T, filepath string) {
				assert.FileExists(t, filepath)

				savedBytes, err := os.ReadFile(filepath)
				require.NoError(t, err, "Failed to read the newly saved file")

				var savedConfig Config
				err = yaml.Unmarshal(savedBytes, &savedConfig)
				require.NoError(t, err, "Saved file content should be valid YAML")

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
				assert.NoFileExists(t, filepath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &ConfigStore{
				filepath: tt.fields.filepath,
				config:   tt.fields.config,
			}

			err := cs.SaveToFile()
			tt.wantErr(t, err, fmt.Sprintf("SaveToFile() with filepath %s", tt.fields.filepath))

			if tt.postCheck != nil {
				tt.postCheck(t, tt.fields.filepath)
			}
		})
	}
}

func TestConfigStore_GetFilepath(t *testing.T) {
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
			cs := &ConfigStore{
				filepath: tt.fields.filepath,
				config:   tt.fields.config,
			}
			assert.Equalf(t, tt.want, cs.GetFilepath(), "GetFilepath()")
		})
	}
}

func TestGenerateSchema(t *testing.T) {
	// Verify the generated schema catches validation errors
	invalidConfig := &Config{
		Clusters: []Cluster{
			{
				Name: "",
			},
		},
	}

	tests := []struct {
		name          string
		config        *Config
		wantErr       bool
		shouldBeValid bool
	}{
		{
			name:          "Generated schema validates a valid config",
			config:        newValidTestConfig(),
			wantErr:       false,
			shouldBeValid: true,
		},
		{
			name:          "Generated schema rejects an invalid config",
			config:        invalidConfig,
			wantErr:       false,
			shouldBeValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaDoc, err := GenerateSchema()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, schemaDoc)

			schemaJSON, err := json.Marshal(schemaDoc)
			require.NoError(t, err)
			require.NotEmpty(t, schemaJSON)

			// Compile and test the generated schema
			const schemaURL = "mem://config.schema.json"
			c := schemaValidator.NewCompiler()
			c.AssertFormat()
			err = c.AddResource(schemaURL, schemaDoc)
			require.NoError(t, err)

			compiled, err := c.Compile(schemaURL)
			require.NoError(t, err)

			var instance any
			data, err := json.Marshal(tt.config)
			require.NoError(t, err)
			err = json.Unmarshal(data, &instance)
			require.NoError(t, err)

			err = compiled.Validate(instance)
			if tt.shouldBeValid {
				assert.NoError(t, err, "Schema should validate valid config")
			} else {
				assert.Error(t, err, "Schema should reject invalid config")
			}
		})
	}
}

func TestGenerateSchema_ComposesCatalogServiceKeys(t *testing.T) {
	schemaDoc, err := GenerateSchema()
	require.NoError(t, err)

	defs, ok := schemaDoc["$defs"].(map[string]any)
	require.True(t, ok)

	servicesDef, ok := defs["Services"].(map[string]any)
	require.True(t, ok)

	properties, ok := servicesDef["properties"].(map[string]any)
	require.True(t, ok)

	assert.Contains(t, properties, "cert-manager")
	assert.Contains(t, properties, "argocd")
	assert.Contains(t, properties, "metallb")
}

func TestLoadAndValidate_MinimalConfigWithDefaults(t *testing.T) {
	// A minimal YAML that only provides required fields and omits all fields
	// that have defaults. After Load() applies defaults, Validate() must pass.
	minimalYAML := `
clusters:
  - name: minimal-cluster
    dnsName: minimal.example.com
    argocd:
      repo:
        https:
          customer:
            url: "https://github.com/customer/repo.git"
          managed:
            url: "https://github.com/managed/repo.git"
    services:
      argocd: {}
      cert-manager:
        config:
          clusterIssuer:
            email: cert@example.com
      external-dns: {}
      external-secrets: {}
      kube-prometheus-stack: {}
      traefik: {}
      kyverno: {}
      kyverno-policies: {}
      kyverno-policy-reporter: {}
      loki: {}
      homer-dashboard: {}
      oauth2-proxy: {}
      metrics-server: {}
      metallb: {}
      longhorn: {}
`

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(minimalYAML), 0644))

	cs := NewConfigStore(configPath)
	require.NoError(t, cs.Load(), "Load should succeed")

	c := cs.GetConfig().Clusters[0]
	assert.Equal(t, "dev", c.Stage, "Stage should be defaulted")
	assert.Equal(t, "hub", c.Type, "Type should be defaulted")
	assert.Equal(t, "traefik", c.IngressClassName, "IngressClassName should be defaulted")

	assert.NoError(t, cs.validate(), "Validate should pass after defaults are applied")
}
