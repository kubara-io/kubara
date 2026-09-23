package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/service"

	schemaValidator "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// Helper function to create a valid test config
func newValidTestConfig() *Config {
	return &Config{
		Version:          ConfigVersionV1Alpha4,
		BootstrapCatalog: testBootstrapCatalogPtr(),
		Clusters: []Cluster{
			{
				Name:       "test-cluster",
				Stage:      "dev",
				Networking: &ClusterNetworking{Type: NetworkingIngress, Ingress: &IngressNetworking{ClassName: "traefik"}},
				Type:       "hub",
				DNSName:    "test-cluster.example.com",
				Catalogs:   testClusterCatalogs(),
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
					SelfManaged: ArgoCDSelfManagedEnabled,
					Repo: RepoProto{
						HTTPS: &RepoType{
							Configs: Repository{
								URL:            "https://github.com/example/configs.git",
								TargetRevision: "main",
							},
							Components: Repository{
								URL:            "https://github.com/example/components.git",
								TargetRevision: "main",
							},
						},
					},
				},
				Services: service.Services{
					"cert-manager": {Status: service.StatusEnabled, Config: service.Config{"clusterIssuer": map[string]any{"name": "letsencrypt-prod", "email": "cert@example.com", "server": "https://acme-v02.api.letsencrypt.org/directory"}}},
				},
			},
		},
	}
}

func createLoadedConfigStore(t *testing.T, cfg *Config) *ConfigStore {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configYAML, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, configYAML, 0o644))

	cs := NewConfigStore(tempDir, configPath, catalog.LoadOptions{})
	require.NoError(t, cs.Load())

	return cs
}

func TestValidateProviderKubernetesTypes(t *testing.T) {
	tests := []struct {
		name           string
		provider       TerraformProvider
		kubernetesType string
		wantErr        bool
	}{
		{name: "stackit supports ske", provider: TerraformProviderStackit, kubernetesType: "ske"},
		{name: "stackit supports edge", provider: TerraformProviderStackit, kubernetesType: "edge"},
		{name: "t-cloud-public supports cce", provider: TerraformProviderTCloudPublic, kubernetesType: "cce"},
		{name: "stackit rejects cce", provider: TerraformProviderStackit, kubernetesType: "cce", wantErr: true},
		{name: "t-cloud-public rejects ske", provider: TerraformProviderTCloudPublic, kubernetesType: "ske", wantErr: true},
		{name: "t-cloud-public rejects edge", provider: TerraformProviderTCloudPublic, kubernetesType: "edge", wantErr: true},
		{name: "unknown provider is ignored by combination validation", provider: TerraformProvider("unknown"), kubernetesType: "ske"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidTestConfig()
			cfg.Clusters[0].Terraform.Provider = tt.provider
			cfg.Clusters[0].Terraform.KubernetesType = tt.kubernetesType

			err := validateProviderKubernetesTypes(cfg)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "terraform.provider")
				assert.Contains(t, err.Error(), "terraform.kubernetesType")
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestConfigStore_LoadNetworking(t *testing.T) {
	gateway := &service.GatewayReference{Name: "edge", Namespace: "networking"}
	for _, tt := range []struct {
		name           string
		networking     *ClusterNetworking
		serviceGateway *service.GatewayReference
		wantErr        string
	}{
		{name: "omitted networking"},
		{name: "empty networking", networking: &ClusterNetworking{}},
		{name: "custom ingress", networking: &ClusterNetworking{Type: NetworkingIngress, Ingress: &IngressNetworking{ClassName: "custom"}}},
		{name: "unknown type", networking: &ClusterNetworking{Type: "unknown"}, wantErr: "networking/type"},
		{name: "gateway", networking: &ClusterNetworking{Type: NetworkingGateway, Gateway: gateway}},
		{name: "gateway listener", networking: &ClusterNetworking{Type: NetworkingGateway, Gateway: &service.GatewayReference{Name: "edge", Namespace: "networking", SectionName: "https"}}},
		{name: "missing gateway", networking: &ClusterNetworking{Type: NetworkingGateway}, wantErr: "networking.gateway is required"},
		{name: "missing gateway name", networking: &ClusterNetworking{Type: NetworkingGateway, Gateway: &service.GatewayReference{Namespace: "networking"}}, wantErr: "name"},
		{name: "missing gateway namespace", networking: &ClusterNetworking{Type: NetworkingGateway, Gateway: &service.GatewayReference{Name: "edge"}}, wantErr: "namespace"},
		{name: "ingress selected with both blocks", networking: &ClusterNetworking{Type: NetworkingIngress, Ingress: &IngressNetworking{ClassName: "custom"}, Gateway: gateway}},
		{name: "gateway selected with both blocks", networking: &ClusterNetworking{Type: NetworkingGateway, Ingress: &IngressNetworking{ClassName: "custom"}, Gateway: gateway}},
		{name: "gateway presence does not select type", networking: &ClusterNetworking{Gateway: gateway}},
		{name: "service gateway override", networking: &ClusterNetworking{Type: NetworkingGateway, Gateway: gateway}, serviceGateway: gateway},
		{name: "incomplete service gateway", networking: &ClusterNetworking{Type: NetworkingGateway, Gateway: gateway}, serviceGateway: &service.GatewayReference{Name: "private"}, wantErr: "namespace"},
		{name: "service gateway with ingress selected", networking: &ClusterNetworking{Type: NetworkingIngress, Gateway: gateway}, serviceGateway: gateway, wantErr: "requires networking.type gateway"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidTestConfig()
			cfg.Clusters[0].Networking = tt.networking
			svc := cfg.Clusters[0].Services["cert-manager"]
			svc.Networking = &service.Networking{Gateway: tt.serviceGateway}
			cfg.Clusters[0].Services["cert-manager"] = svc
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			data, err := yaml.Marshal(cfg)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, data, 0600))
			store := NewConfigStore(dir, path, catalog.LoadOptions{})
			err = store.Load()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			loaded := store.GetConfig().Clusters[0]
			wantType := NetworkingIngress
			if tt.networking != nil && tt.networking.Type != "" {
				wantType = tt.networking.Type
			}
			assert.Equal(t, wantType, loaded.Networking.Type)
			// Catalog serialization retains the legacy input only when an Ingress block exists.
			instance := configInstance(t, store.GetConfig())["clusters"].([]any)[0].(map[string]any)
			if loaded.Networking.Ingress != nil {
				assert.Equal(t, loaded.Networking.Ingress.ClassName, instance["ingressClassName"])
			} else {
				assert.NotContains(t, instance, "ingressClassName")
			}
			assert.Equal(t, tt.serviceGateway, loaded.Services["cert-manager"].Networking.Gateway)
			require.NoError(t, store.SaveToFile())
			saved, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.NotContains(t, string(saved), "ingressClassName:")
			assert.Equal(t, loaded.IngressClassName, store.GetConfig().Clusters[0].IngressClassName)
			require.NoError(t, store.Load())
			assert.Equal(t, loaded.Networking, store.GetConfig().Clusters[0].Networking)
		})
	}
}

func TestConfigStore_MigrateIngressClassName(t *testing.T) {
	cfg := newValidTestConfig()
	cfg.Version = ConfigVersionV1Alpha4
	cfg.Clusters[0].Networking = nil
	cfg.Clusters[0].IngressClassName = "custom"
	store := createLoadedConfigStore(t, cfg)
	cluster := store.GetConfig().Clusters[0]
	assert.Equal(t, ConfigVersionV1Alpha4, store.GetConfig().Version)
	assert.Equal(t, NetworkingIngress, cluster.Networking.Type)
	assert.Equal(t, "custom", cluster.Networking.Ingress.ClassName)
	assert.Equal(t, "custom", cluster.IngressClassName)
	saved, err := os.ReadFile(store.GetFilepath())
	require.NoError(t, err)
	assert.NotContains(t, string(saved), "ingressClassName:")
	require.NoError(t, store.Load())
	assert.Equal(t, "custom", store.GetConfig().Clusters[0].IngressClassName)

	// Reintroducing even the same legacy value after migration must fail.
	var raw map[string]any
	require.NoError(t, yaml.Unmarshal(saved, &raw))
	raw["clusters"].([]any)[0].(map[string]any)["ingressClassName"] = "custom"
	edited, err := yaml.Marshal(raw)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(store.GetFilepath(), edited, 0600))
	require.ErrorContains(t, store.Load(), "ingressClassName is no longer supported")
	unchanged, err := os.ReadFile(store.GetFilepath())
	require.NoError(t, err)
	assert.Equal(t, edited, unchanged)
}

func TestConfigStore_RejectReintroducedIngressClassName(t *testing.T) {
	for _, value := range []any{"custom", "traefik", "", nil} {
		t.Run(fmt.Sprintf("value_%v", value), func(t *testing.T) {
			cfg := newValidTestConfig()
			instance := configInstance(t, cfg)
			instance["clusters"].([]any)[0].(map[string]any)["ingressClassName"] = value
			data, err := yaml.Marshal(instance)
			require.NoError(t, err)
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(path, data, 0600))
			store := NewConfigStore(dir, path, catalog.LoadOptions{})
			require.ErrorContains(t, store.Load(), "ingressClassName is no longer supported")
			unchanged, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, data, unchanged)
		})
	}
}

// Helper function to deep copy a config
func deepCopyConfig(c *Config) *Config {
	newConfig := *c
	newConfig.Clusters = make([]Cluster, len(c.Clusters))
	copy(newConfig.Clusters, c.Clusters)
	return &newConfig
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
			cs := NewConfigStore(".", tt.filepath, testCatalogLoadOptions())
			err := cs.Load()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tt.wantConfig.Clusters[0].IngressClassName = tt.wantConfig.Clusters[0].Networking.Ingress.ClassName
				assert.Equal(t, tt.wantConfig, cs.GetConfig())
			}
		})
	}
}

func TestConfigStore_LoadRejectsLegacyMigrationConflicts(t *testing.T) {
	tests := []struct {
		name        string
		servicesYML string
		wantErrs    []string
	}{
		{
			name: "duplicate canonical service names",
			servicesYML: `
      certManager:
        status: enabled
      cert-manager:
        status: enabled
`,
			wantErrs: []string{
				"conflicting keys",
				`"certManager"`,
				`"cert-manager"`,
				`canonical service "cert-manager"`,
			},
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
			wantErrs: []string{"both legacy clusterIssuer and config.clusterIssuer"},
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
			wantErrs: []string{"both legacy storageClassName and storage.className"},
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
			wantErrs: []string{"both legacy ingress.annotations and networking.annotations"},
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
          configs:
            url: "https://github.com/customer/repo.git"
          components:
            url: "https://github.com/managed/repo.git"
    services:%s`, tt.servicesYML)

			configPath := filepath.Join(t.TempDir(), "legacy-conflict.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(legacyYAML), 0644))

			cs := NewConfigStore(".", configPath, testCatalogLoadOptions())
			err := cs.Load()
			require.Error(t, err)
			for _, wantErr := range tt.wantErrs {
				assert.ErrorContains(t, err, wantErr)
			}
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

	invalidConfigProviderEnumMismatch := deepCopyConfig(validConfig)
	clonedTerraformProvider := *invalidConfigProviderEnumMismatch.Clusters[0].Terraform
	clonedTerraformProvider.Provider = TerraformProvider("unknown")
	invalidConfigProviderEnumMismatch.Clusters[0].Terraform = &clonedTerraformProvider

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
			name:    "invalid_config_should_fail_on_provider_enum_mismatch",
			config:  invalidConfigProviderEnumMismatch,
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
				Name:       "prod-cluster",
				Networking: &ClusterNetworking{Type: NetworkingIngress, Ingress: &IngressNetworking{ClassName: "traefik"}},
				Stage:      "production",
				Type:       "hub",
				DNSName:    "prod.example.com",
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
			cs := createLoadedConfigStore(t, newValidTestConfig())
			schemaDoc, err := cs.GenerateSchema()
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

func TestGenerateSchema_TerraformProviderNoneAllowsMissingTerraformDetails(t *testing.T) {
	cs := createLoadedConfigStore(t, newValidTestConfig())
	schemaDoc, err := cs.GenerateSchema()
	require.NoError(t, err)

	const schemaURL = "mem://config.schema.json"
	c := schemaValidator.NewCompiler()
	c.AssertFormat()
	require.NoError(t, c.AddResource(schemaURL, schemaDoc))

	compiled, err := c.Compile(schemaURL)
	require.NoError(t, err)

	validNoneConfig := configInstance(t, newValidTestConfig())
	cluster := validNoneConfig["clusters"].([]any)[0].(map[string]any)
	cluster["terraform"] = map[string]any{
		"provider": "none",
	}
	assert.NoError(t, compiled.Validate(validNoneConfig))

	invalidStackitConfig := configInstance(t, newValidTestConfig())
	cluster = invalidStackitConfig["clusters"].([]any)[0].(map[string]any)
	cluster["terraform"] = map[string]any{
		"provider": "stackit",
	}
	assert.Error(t, compiled.Validate(invalidStackitConfig))
}

func configInstance(t *testing.T, cfg *Config) map[string]any {
	t.Helper()

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	var instance map[string]any
	require.NoError(t, json.Unmarshal(data, &instance))
	return instance
}

func TestGenerateSchema_ComposesCatalogServiceKeys(t *testing.T) {
	cs := createLoadedConfigStore(t, newValidTestConfig())
	schemaDoc, err := cs.GenerateSchema()
	require.NoError(t, err)

	defs, ok := schemaDoc["$defs"].(map[string]any)
	require.True(t, ok)

	servicesDef, ok := defs["Services"].(map[string]any)
	require.True(t, ok)

	properties, ok := servicesDef["properties"].(map[string]any)
	require.True(t, ok)

	assert.Contains(t, properties, "cert-manager")
	assert.NotContains(t, properties, "argocd")
	assert.NotContains(t, properties, "argo-cd")
	assert.NotContains(t, properties, "bootstrap-crds")
}

func TestConfigStore_LoadAppliesDefaultsPerClusterCatalog(t *testing.T) {
	cfg := newValidTestConfig()
	cfg.Clusters[0].Catalogs = []string{testGeneralCatalogPath}
	cfg.Clusters[0].Services = service.Services{}

	secondCluster := cfg.Clusters[0]
	secondCluster.Name = "logging-cluster"
	secondCluster.DNSName = "logging.example.com"
	secondCluster.Catalogs = []string{testCustomCatalogPath}
	secondCluster.Services = service.Services{}
	cfg.Clusters = append(cfg.Clusters, secondCluster)

	cs := createLoadedConfigStore(t, cfg)

	require.Len(t, cs.GetConfig().Clusters, 2)
	assert.Contains(t, cs.GetConfig().Clusters[0].Services, "cert-manager")
	assert.NotContains(t, cs.GetConfig().Clusters[0].Services, "loki")
	assert.Contains(t, cs.GetConfig().Clusters[1].Services, "loki")
	assert.NotContains(t, cs.GetConfig().Clusters[1].Services, "cert-manager")
}

func TestGenerateSchema_UsesClusterSpecificServiceBranches(t *testing.T) {
	cfg := newValidTestConfig()
	cfg.Clusters[0].Catalogs = []string{testGeneralCatalogPath}
	cfg.Clusters[0].Services = service.Services{}

	secondCluster := cfg.Clusters[0]
	secondCluster.Name = "logging-cluster"
	secondCluster.DNSName = "logging.example.com"
	secondCluster.Catalogs = []string{testCustomCatalogPath}
	secondCluster.Services = service.Services{}
	cfg.Clusters = append(cfg.Clusters, secondCluster)

	cs := createLoadedConfigStore(t, cfg)
	schemaDoc, err := cs.GenerateSchema()
	require.NoError(t, err)

	defs, ok := schemaDoc["$defs"].(map[string]any)
	require.True(t, ok)
	clusterDef, ok := defs["Cluster"].(map[string]any)
	require.True(t, ok)
	branches, ok := clusterDef["oneOf"].([]any)
	require.True(t, ok)
	require.Len(t, branches, 2)

	servicesByCluster := make(map[string]map[string]any, len(branches))
	for _, rawBranch := range branches {
		properties := rawBranch.(map[string]any)["properties"].(map[string]any)
		name := properties["name"].(map[string]any)["const"].(string)
		services := properties["services"].(map[string]any)["properties"].(map[string]any)
		servicesByCluster[name] = services
	}
	assert.Contains(t, servicesByCluster["test-cluster"], "cert-manager")
	assert.NotContains(t, servicesByCluster["test-cluster"], "loki")
	assert.Contains(t, servicesByCluster["logging-cluster"], "loki")
	assert.NotContains(t, servicesByCluster["logging-cluster"], "cert-manager")
}

func TestLoadAndValidate_MinimalConfigWithDefaults(t *testing.T) {
	// A minimal YAML that only provides required fields and omits all fields
	// that have defaults. After Load() applies defaults, Validate() must pass.
	minimalYAML := fmt.Sprintf(`
clusters:
  - name: minimal-cluster
    dnsName: minimal.example.com
    argocd:
      repo:
        https:
          configs:
            url: "https://github.com/customer/repo.git"
          components:
            url: "https://github.com/managed/repo.git"
    catalogs:
      - %q
    services:
      cert-manager:
        config:
          clusterIssuer:
            email: cert@example.com
`, testGeneralCatalogPath)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(minimalYAML), 0644))

	cs := NewConfigStore(".", configPath, testCatalogLoadOptions())
	require.NoError(t, cs.Load(), "Load should succeed")

	c := cs.GetConfig().Clusters[0]
	assert.Equal(t, "dev", c.Stage, "Stage should be defaulted")
	assert.Equal(t, "hub", c.Type, "Type should be defaulted")
	assert.Equal(t, "traefik", c.Networking.Ingress.ClassName, "IngressClassName should be defaulted")

}

func TestConfigStore_LoadStripsBootstrapServicesFromV1Alpha4Clusters(t *testing.T) {
	configYAML := fmt.Sprintf(`
version: %s
bootstrapCatalog: %q
clusters:
  - name: migrated-cluster
    stage: dev
    type: hub
    dnsName: migrated.example.com
    ingressClassName: traefik
    catalogs:
      - %q
    argocd:
      repo:
        https:
          configs:
            url: "https://github.com/example/configs.git"
            targetRevision: main
          components:
            url: "https://github.com/example/components.git"
            targetRevision: main
    services:
      argo-cd:
        status: enabled
      bootstrap-crds:
        status: disabled
      cert-manager:
        status: enabled
        config:
          clusterIssuer:
            name: letsencrypt-prod
            email: cert@example.com
            server: https://acme-v02.api.letsencrypt.org/directory
`, ConfigVersionV1Alpha4, *testBootstrapCatalogPtr(), testGeneralCatalogPath)

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0644))

	cs := NewConfigStore(".", configPath, testCatalogLoadOptions())
	require.NoError(t, cs.Load())

	cluster := cs.GetConfig().Clusters[0]
	assert.NotContains(t, cluster.Services, "argo-cd")
	assert.NotContains(t, cluster.Services, "bootstrap-crds")
	assert.Contains(t, cluster.Services, "cert-manager")

	require.NoError(t, cs.SaveToFile())

	savedBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var savedConfig Config
	require.NoError(t, yaml.Unmarshal(savedBytes, &savedConfig))
	require.Len(t, savedConfig.Clusters, 1)
	assert.NotContains(t, savedConfig.Clusters[0].Services, "argo-cd")
	assert.NotContains(t, savedConfig.Clusters[0].Services, "bootstrap-crds")
	assert.Contains(t, savedConfig.Clusters[0].Services, "cert-manager")
}

func TestLoadAndValidate_TerraformProviderNoneDisablesTerraform(t *testing.T) {
	configYAML := fmt.Sprintf(`
version: %s
bootstrapCatalog: %q
clusters:
  - name: helm-only-cluster
    dnsName: helm-only.example.com
    catalogs:
      - %q
    terraform:
      provider: none
    argocd:
      repo:
        https:
          configs:
            url: "https://github.com/customer/repo.git"
          components:
            url: "https://github.com/managed/repo.git"
`, ConfigVersionV1Alpha4, *testBootstrapCatalogPtr(), testGeneralCatalogPath)

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(configYAML), 0644))

	cs := NewConfigStore(".", configPath, testCatalogLoadOptions())
	require.NoError(t, cs.Load())

	require.Len(t, cs.GetConfig().Clusters, 1)
	assert.Nil(t, cs.GetConfig().Clusters[0].Terraform)
}

func TestGenerateSchema_Networking(t *testing.T) {
	store := createLoadedConfigStore(t, newValidTestConfig())
	schema, err := store.GenerateSchema()
	require.NoError(t, err)
	for _, tt := range []struct {
		name    string
		field   string
		value   any
		wantErr bool
	}{
		{name: "ingress", field: "networking", value: map[string]any{"type": "ingress", "ingress": map[string]any{"className": "custom"}}},
		{name: "gateway", field: "networking", value: map[string]any{"type": "gateway", "gateway": map[string]any{"name": "edge", "namespace": "networking"}}},
		{name: "invalid type", field: "networking", value: map[string]any{"type": "unknown"}, wantErr: true},
		{name: "legacy field", field: "ingressClassName", value: "custom", wantErr: true},
		{name: "old nested field", field: "networking", value: map[string]any{"ingressClassName": "custom"}, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			instance := configInstance(t, newValidTestConfig())
			instance["clusters"].([]any)[0].(map[string]any)[tt.field] = tt.value
			err := validateAgainstSchema(schema, instance)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfigStore_LoadRejectsUnknownNetworkingFields(t *testing.T) {
	for _, tt := range []struct {
		name              string
		networking        map[string]any
		serviceNetworking map[string]any
		wantErr           string
	}{
		{name: "old nested ingress class", networking: map[string]any{"type": "ingress", "ingressClassName": "custom"}, wantErr: "ingressClassName"},
		{name: "null old nested ingress class", networking: map[string]any{"ingressClassName": nil}, wantErr: "ingressClassName"},
		{name: "ingress typo", networking: map[string]any{"ingress": map[string]any{"classNam": "custom"}}, wantErr: "classNam"},
		{name: "gateway typo", networking: map[string]any{"type": "gateway", "gateway": map[string]any{"name": "edge", "namespace": "networking", "sectionNam": "https"}}, wantErr: "sectionNam"},
		{name: "service networking typo", serviceNetworking: map[string]any{"annotation": map[string]any{"example.com/key": "value"}}, wantErr: "annotation"},
		{name: "service gateway typo", networking: map[string]any{"type": "gateway", "gateway": map[string]any{"name": "edge", "namespace": "networking"}}, serviceNetworking: map[string]any{"gateway": map[string]any{"name": "edge", "namespace": "networking", "sectionNam": "https"}}, wantErr: "sectionNam"},
		{name: "valid ingress and arbitrary annotations", networking: map[string]any{"type": "ingress", "ingress": map[string]any{"className": "custom"}}, serviceNetworking: map[string]any{"annotations": map[string]any{"example.com/key": "value"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			raw := configInstance(t, newValidTestConfig())
			cluster := raw["clusters"].([]any)[0].(map[string]any)
			if tt.networking != nil {
				cluster["networking"] = tt.networking
			}
			if tt.serviceNetworking != nil {
				svc := cluster["services"].(map[string]any)["cert-manager"].(map[string]any)
				svc["networking"] = tt.serviceNetworking
			}
			data, err := yaml.Marshal(raw)
			require.NoError(t, err)
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			require.NoError(t, os.WriteFile(path, data, 0600))
			store := NewConfigStore(dir, path, catalog.LoadOptions{})
			err = store.Load()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, "unknown networking field")
				require.ErrorContains(t, err, tt.wantErr)
				unchanged, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, data, unchanged)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "custom", store.GetConfig().Clusters[0].Networking.Ingress.ClassName)
			assert.Equal(t, "custom", store.GetConfig().Clusters[0].IngressClassName)
		})
	}
}

func TestConfigStore_LoadValidatesBeforePopulatingCompatibilityFields(t *testing.T) {
	cfg := newValidTestConfig()
	cfg.Clusters[0].DNSName = "not a valid hostname"
	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, data, 0600))
	store := NewConfigStore(dir, path, catalog.LoadOptions{})
	require.ErrorContains(t, store.Load(), "validate config")
	assert.Empty(t, store.GetConfig().Clusters[0].IngressClassName)
}
