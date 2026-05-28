package render

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/kubara-io/kubara/internal/catalog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testTemplatesFS = catalog.BuiltInFS()

// helper function to setup test filesystem with correct root path
func setupTestFS(_ *testing.T) func() {
	originalFS := templatesFSNew
	templatesFSNew = testTemplatesFS

	// Return cleanup function
	return func() {
		templatesFSNew = originalFS
	}
}

func fullServiceContext() map[string]any {
	return map[string]any{
		"argocd":                  map[string]any{"status": "enabled"},
		"cert-manager":            map[string]any{"status": "enabled", "config": map[string]any{"clusterIssuer": map[string]any{"name": "letsencrypt-prod", "email": "admin@example.com", "server": "https://acme-staging-v02.api.letsencrypt.org/directory"}}},
		"external-dns":            map[string]any{"status": "enabled"},
		"external-secrets":        map[string]any{"status": "enabled"},
		"kube-prometheus-stack":   map[string]any{"status": "enabled"},
		"traefik":                 map[string]any{"status": "enabled"},
		"kyverno":                 map[string]any{"status": "enabled"},
		"kyverno-policies":        map[string]any{"status": "enabled"},
		"kyverno-policy-reporter": map[string]any{"status": "enabled"},
		"loki":                    map[string]any{"status": "enabled"},
		"homer-dashboard":         map[string]any{"status": "enabled"},
		"oauth2-proxy":            map[string]any{"status": "disabled"},
		"metrics-server":          map[string]any{"status": "disabled"},
		"metallb":                 map[string]any{"status": "disabled"},
		"longhorn":                map[string]any{"status": "disabled"},
	}
}

func fullCatalogContext() map[string]any {
	return map[string]any{
		"services": map[string]any{
			"argocd":                  map[string]any{"chartPath": "argo-cd"},
			"cert-manager":            map[string]any{"chartPath": "cert-manager"},
			"external-dns":            map[string]any{"chartPath": "external-dns"},
			"external-secrets":        map[string]any{"chartPath": "external-secrets"},
			"kube-prometheus-stack":   map[string]any{"chartPath": "kube-prometheus-stack"},
			"traefik":                 map[string]any{"chartPath": "traefik"},
			"kyverno":                 map[string]any{"chartPath": "kyverno"},
			"kyverno-policies":        map[string]any{"chartPath": "kyverno-policies"},
			"kyverno-policy-reporter": map[string]any{"chartPath": "kyverno-policy-reporter"},
			"loki":                    map[string]any{"chartPath": "loki"},
			"homer-dashboard":         map[string]any{"chartPath": "homer-dashboard"},
			"oauth2-proxy":            map[string]any{"chartPath": "oauth2-proxy"},
			"metrics-server":          map[string]any{"chartPath": "metrics-server"},
			"metallb":                 map[string]any{"chartPath": "metallb"},
			"longhorn":                map[string]any{"chartPath": "longhorn"},
		},
	}
}

func TestTemplateType_String(t *testing.T) {
	tests := []struct {
		name string
		tt   TemplateType
		want string
	}{
		{
			name: "Terraform type returns correct string",
			tt:   Terraform,
			want: "terraform",
		},
		{
			name: "Helm type returns correct string",
			tt:   Helm,
			want: "helm",
		},
		{
			name: "All type returns correct string",
			tt:   All,
			want: "all",
		},
		{
			name: "Invalid type returns empty string",
			tt:   TemplateType(99),
			want: "", // Falls back to empty since not in map
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.tt.String())
		})
	}
}

func TestMakeWalkDirFunc(t *testing.T) {
	// Create test filesystem structure
	testFS := testTemplatesFS

	var files []string
	walkFunc := makeWalkDirFunc(tmplRoot, &files)

	err := fs.WalkDir(testFS, tmplRoot, walkFunc)
	require.NoError(t, err)

	// Verify that files are collected (not directories)
	require.NotEmpty(t, files)
	for _, file := range files {
		assert.NotEmpty(t, file)
		assert.False(t, strings.HasSuffix(file, "/"))
	}

	// Test error propagation if WalkDir encounters an error
	var errorFiles []string
	errorWalkFunc := makeWalkDirFunc(tmplRoot, &errorFiles)
	// Intentionally walk non-existent path to trigger error
	err = fs.WalkDir(testFS, "nonexistent", errorWalkFunc)
	assert.Error(t, err)
	assert.Empty(t, errorFiles)
}

func TestMakeWalkDirFunc_RelPathError(t *testing.T) {
	// Test relative path error (edge case: path outside root)
	testFS := testTemplatesFS
	var files []string
	walkFunc := makeWalkDirFunc("nonexistent-root", &files) // Invalid root

	err := fs.WalkDir(testFS, tmplRoot, walkFunc)
	// Should still work but paths might be relative to nonexistent root
	require.NoError(t, err)
}

func TestMakeWalkDirFunc_DirectoryFiltering(t *testing.T) {
	// Test that directories are properly filtered out
	testFS := testTemplatesFS
	var files []string
	walkFunc := makeWalkDirFunc(tmplRoot, &files)

	err := fs.WalkDir(testFS, tmplRoot, walkFunc)
	require.NoError(t, err)

	// Ensure no directory entries (ending with /) are included
	for _, file := range files {
		assert.False(t, strings.HasSuffix(file, "/"), "File path should not end with /: %s", file)
	}
}

func TestSelectTemplatesForProvider_PrefersProviderSpecificFile(t *testing.T) {
	files := []string{
		"managed-service-catalog/terraform/modules/iam/main.tf",
		"managed-service-catalog/terraform/providers/stackit/modules/iam/main.tf",
		"managed-service-catalog/terraform/providers/t-cloud-public/modules/iam/main.tf",
		"managed-service-catalog/terraform/modules/iam/variables.tf",
	}

	selected := selectTemplatesForProvider(files, "stackit")

	assert.Contains(t, selected, "managed-service-catalog/terraform/providers/stackit/modules/iam/main.tf")
	assert.NotContains(t, selected, "managed-service-catalog/terraform/modules/iam/main.tf")
	assert.NotContains(t, selected, "managed-service-catalog/terraform/providers/t-cloud-public/modules/iam/main.tf")
	assert.Contains(t, selected, "managed-service-catalog/terraform/modules/iam/variables.tf")
	require.Len(t, selected, 2)
}

func TestSelectTemplatesForProvider_FallsBackToCommonFile(t *testing.T) {
	files := []string{
		"managed-service-catalog/terraform/modules/iam/main.tf",
		"managed-service-catalog/terraform/providers/stackit/modules/iam/main.tf",
		"managed-service-catalog/terraform/modules/iam/variables.tf",
	}

	selected := selectTemplatesForProvider(files, "azure")

	assert.Contains(t, selected, "managed-service-catalog/terraform/modules/iam/main.tf")
	assert.NotContains(t, selected, "managed-service-catalog/terraform/providers/stackit/modules/iam/main.tf")
	assert.Contains(t, selected, "managed-service-catalog/terraform/modules/iam/variables.tf")
	require.Len(t, selected, 2)
}

func TestStripProviderPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strips providers/stackit under terraform",
			input: "customer-service-catalog/terraform/providers/stackit/example/infrastructure/main.tf.tplt",
			want:  "customer-service-catalog/terraform/example/infrastructure/main.tf.tplt",
		},
		{
			name:  "strips providers/stackit under managed terraform",
			input: "managed-service-catalog/terraform/providers/stackit/modules/ske-cluster/main.tf",
			want:  "managed-service-catalog/terraform/modules/ske-cluster/main.tf",
		},
		{
			name:  "leaves non-provider terraform path unchanged",
			input: "managed-service-catalog/terraform/images/public-cloud-0.png",
			want:  "managed-service-catalog/terraform/images/public-cloud-0.png",
		},
		{
			name:  "does not strip providers/<name> outside terraform or helm context",
			input: "some-catalog/providers/stackit/file.txt",
			want:  "some-catalog/providers/stackit/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StripProviderPath(tt.input))
		})
	}
}

func TestTemplateFiles_TCloudPublicProviderSelectsCCEArtifacts(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var paths []string
	for _, result := range results {
		require.NoError(t, result.Error)
		paths = append(paths, result.Path)
	}

	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/cce-cluster/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/network/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/dns-zone/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/keypair/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/kms-key/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/identity-agencies/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/objectstorage-bucket/main.tf")
	assert.Contains(t, paths, "customer-service-catalog/terraform/providers/t-cloud-public/example/bootstrap-tfstate-backend/main.tf.tplt")
	assert.NotContains(t, paths, "managed-service-catalog/terraform/providers/stackit/modules/ske-cluster/main.tf")
}

func TestTemplateFiles_TCloudPublicAgenciesUseDefaultProvider(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var bootstrapMain string
	var infrastructureMain string
	var infrastructureProviders string
	var agencyMain string
	var agencyVariables string
	for _, result := range results {
		require.NoError(t, result.Error)
		switch result.Path {
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/bootstrap-tfstate-backend/main.tf.tplt":
			bootstrapMain = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/main.tf.tplt":
			infrastructureMain = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/terraform.tf.tplt":
			infrastructureProviders = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/identity-agencies/main.tf":
			agencyMain = result.Content
		case "managed-service-catalog/terraform/providers/t-cloud-public/modules/identity-agencies/variables.tf":
			agencyVariables = result.Content
		}
	}

	require.NotEmpty(t, bootstrapMain)
	require.NotEmpty(t, infrastructureMain)
	require.NotEmpty(t, infrastructureProviders)
	require.NotEmpty(t, agencyMain)
	require.NotEmpty(t, agencyVariables)

	for _, content := range []string{bootstrapMain, infrastructureMain, infrastructureProviders} {
		assert.NotContains(t, content, `alias       = "agency"`)
		assert.NotContains(t, content, "opentelekomcloud.agency")
	}
	for _, content := range []string{bootstrapMain, infrastructureMain} {
		assert.NotContains(t, content, "providers = {")
	}
	assert.Contains(t, infrastructureProviders, "tenant_name = var.t_cloud_public_tenant_name")
	assert.Contains(t, agencyMain, "domain_roles          = try(length(each.value.domain_roles), 0) > 0 ? each.value.domain_roles : null")
	assert.Contains(t, agencyVariables, "domain_roles = optional(list(string))")
	assert.NotContains(t, agencyVariables, "domain_roles = optional(list(string), [])")
}

func TestTemplateFiles_TCloudPublicProviderRendersVeleroBucketWhenEnabled(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{
					"velero": map[string]any{"status": "enabled"},
				},
			},
		},
	})

	require.NoError(t, err)

	var mainContent string
	var envContent string
	for _, result := range results {
		require.NoError(t, result.Error)
		switch result.Path {
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/main.tf.tplt":
			mainContent = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/env.auto.tfvars.tplt":
			envContent = result.Content
		}
	}

	require.NotEmpty(t, mainContent)
	require.NotEmpty(t, envContent)
	assert.Contains(t, mainContent, `module "velero_bucket"`)
	assert.Contains(t, envContent, "velero_bucket_name")
}

func TestTemplateFiles_TCloudPublicEnvAutoTfvarsDoesNotRenderProviderCredentials(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":  "test-cluster",
				"stage": "dev",
				"terraform": map[string]any{
					"projectId":         "test-tenant",
					"kubernetesType":    "cce",
					"kubernetesVersion": "1.29",
					"dns":               map[string]any{"name": "example.com", "email": "admin@example.com"},
				},
				"services": map[string]any{},
			},
		},
	})

	require.NoError(t, err)

	var infrastructureEnv string
	var bootstrapEnv string
	var infrastructureVariables string
	for _, result := range results {
		require.NoError(t, result.Error)
		switch result.Path {
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/env.auto.tfvars.tplt":
			infrastructureEnv = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/bootstrap-tfstate-backend/env.auto.tfvars.tplt":
			bootstrapEnv = result.Content
		case "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/variables.tf.tplt":
			infrastructureVariables = result.Content
		}
	}

	require.NotEmpty(t, infrastructureEnv)
	require.NotEmpty(t, bootstrapEnv)
	require.NotEmpty(t, infrastructureVariables)

	for _, content := range []string{infrastructureEnv, bootstrapEnv} {
		assert.NotContains(t, content, "t_cloud_public_region")
		assert.NotContains(t, content, "t_cloud_public_domain_name")
		assert.NotContains(t, content, "t_cloud_public_tenant_name")
		assert.NotContains(t, content, "t_cloud_public_access_key")
		assert.NotContains(t, content, "t_cloud_public_secret_key")
	}
	assert.Contains(t, infrastructureVariables, `default     = "test-tenant"`)
}

func TestTemplateFiles_TCloudPublicProviderOverridesExternalDNSValues(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Helm,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":             "test-cluster",
				"stage":            "dev",
				"dnsName":          "test.example.com",
				"ingressClassName": "traefik",
				"ssoOrg":           "myorg",
				"ssoTeam":          "myteam",
				"services":         fullServiceContext(),
			},
			"catalog": fullCatalogContext(),
		},
	})

	require.NoError(t, err)

	var externalDNSValues string
	var externalDNSPath string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/helm/providers/t-cloud-public/example/external-dns/values.yaml.tplt" {
			externalDNSValues = result.Content
			externalDNSPath = result.Path
		}
	}

	require.NotEmpty(t, externalDNSPath)
	assert.Contains(t, externalDNSValues, "ghcr.io/opentelekomcloud/external-dns-t-cloud-public-webhook")
	assert.Contains(t, externalDNSValues, "tag: 1.1.2")
	assert.Contains(t, externalDNSValues, "secretName: tcloudpubliccloudsyaml")
	assert.Contains(t, externalDNSValues, "key: clouds.yaml")
	assert.NotContains(t, externalDNSValues, "stackit")
}

func TestTemplateFiles_StackitProviderOverridesExternalDNSValues(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type:     Helm,
		Provider: "stackit",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":             "test-cluster",
				"stage":            "dev",
				"dnsName":          "test.example.com",
				"ingressClassName": "traefik",
				"ssoOrg":           "myorg",
				"ssoTeam":          "myteam",
				"services":         fullServiceContext(),
			},
			"catalog": fullCatalogContext(),
		},
	})

	require.NoError(t, err)

	var externalDNSValues string
	var externalDNSPath string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/helm/providers/stackit/example/external-dns/values.yaml.tplt" {
			externalDNSValues = result.Content
			externalDNSPath = result.Path
		}
	}

	require.NotEmpty(t, externalDNSPath)
	assert.Contains(t, externalDNSValues, "ghcr.io/stackitcloud/external-dns-stackit-webhook")
	assert.Contains(t, externalDNSValues, "secretName: external-dns-webhook")
	assert.NotContains(t, externalDNSValues, "opentelekomcloud")
}

func TestTemplateFiles_CommonExternalDNSValuesAreProviderNeutral(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	results, err := TemplateFiles(TemplateOptions{
		Type: Helm,
		Data: map[string]any{
			"cluster": map[string]any{
				"name":             "test-cluster",
				"stage":            "dev",
				"dnsName":          "test.example.com",
				"ingressClassName": "traefik",
				"ssoOrg":           "myorg",
				"ssoTeam":          "myteam",
				"services":         fullServiceContext(),
			},
			"catalog": fullCatalogContext(),
		},
	})

	require.NoError(t, err)

	var externalDNSValues string
	var externalDNSPath string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/helm/example/external-dns/values.yaml.tplt" {
			externalDNSValues = result.Content
			externalDNSPath = result.Path
		}
	}

	require.NotEmpty(t, externalDNSPath)
	assert.Contains(t, externalDNSValues, "externalSecrets: {}")
	assert.NotContains(t, externalDNSValues, "ghcr.io/stackitcloud")
	assert.NotContains(t, externalDNSValues, "ghcr.io/opentelekomcloud")
	assert.NotContains(t, externalDNSValues, "stackit")
}

func TestTemplateFiles_TCloudPublicExternalDNSSkipsExternalSecretWhenDisabled(t *testing.T) {
	cleanup := setupTestFS(t)
	defer cleanup()

	services := fullServiceContext()
	services["external-secrets"] = map[string]any{"status": "disabled"}

	results, err := TemplateFiles(TemplateOptions{
		Type:     Helm,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":             "test-cluster",
				"stage":            "dev",
				"dnsName":          "test.example.com",
				"ingressClassName": "traefik",
				"ssoOrg":           "myorg",
				"ssoTeam":          "myteam",
				"services":         services,
			},
			"catalog": fullCatalogContext(),
		},
	})

	require.NoError(t, err)

	var externalDNSValues string
	for _, result := range results {
		require.NoError(t, result.Error)
		if result.Path == "customer-service-catalog/helm/providers/t-cloud-public/example/external-dns/values.yaml.tplt" {
			externalDNSValues = result.Content
		}
	}

	require.NotEmpty(t, externalDNSValues)
	assert.Contains(t, externalDNSValues, "externalSecrets: {}")
	assert.NotContains(t, externalDNSValues, "remoteKey: t-cloud-public-clouds-yaml")
}

func TestTemplateFiles(t *testing.T) {
	tests := []struct {
		name     string
		tplType  TemplateType
		context  map[string]any
		wantErr  bool
		validate func(t *testing.T, results []TemplateResult)
	}{
		{
			name:    "Success: Successfully template all files of type All",
			tplType: All,
			context: map[string]any{
				"var": map[string]any{
					"project_id": "12345",
					"name":       "test-cluster",
					"stage":      "dev",
				},
				"cluster": map[string]any{
					"type":             "hub",
					"name":             "test-cluster",
					"stage":            "dev",
					"dnsName":          "test.example.com",
					"ingressClassName": "traefik",
					"ssoOrg":           "myorg",
					"ssoTeam":          "myteam",
					"terraform": map[string]any{
						"kubernetesType": "ske",
					},
					"argocd": map[string]any{
						"repo": map[string]any{
							"https": map[string]any{
								"managed": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "managed-service-catalog/helm",
									"targetRevision": "main",
								},
								"customer": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "customer-service-catalog/helm",
									"targetRevision": "main",
								},
							},
						},
						"helmRepo": map[string]any{
							"url": "https://charts.example.com",
						},
					},
					"services": fullServiceContext(),
				},
				"catalog": fullCatalogContext(),
			},
			wantErr: false, // No errors expected with valid context
			validate: func(t *testing.T, results []TemplateResult) {
				assert.NotEmpty(t, results)
				// Should have both template and static files
				hasTemplate := false
				hasStatic := false
				hasValidTemplate := false

				for _, result := range results {
					if strings.HasSuffix(result.Path, ".tplt") {
						hasTemplate = true
						if result.Error == nil {
							hasValidTemplate = true
							assert.NotEmpty(t, result.Content)
						}
					} else {
						hasStatic = true
						assert.NoError(t, result.Error)
						assert.NotEmpty(t, result.Content)
					}
				}
				assert.True(t, hasTemplate, "Should have at least one template file")
				assert.True(t, hasStatic, "Should have at least one static file")
				assert.True(t, hasValidTemplate, "Should have at least one successfully rendered template")
			},
		},
		{
			name:    "Error: Handle template execution errors in all files",
			tplType: All,
			context: map[string]any{},
			wantErr: true,
			validate: func(t *testing.T, results []TemplateResult) {
				assert.NotEmpty(t, results)
				templateFiles := 0
				staticSuccess := 0
				templateErrors := 0
				for _, result := range results {
					if strings.HasSuffix(result.Path, ".tplt") {
						templateFiles++
						if result.Error != nil {
							templateErrors++
						}
					} else {
						if result.Error == nil {
							staticSuccess++
						}
					}
				}
				assert.Greater(t, templateFiles, 0, "Should have template files")
				assert.Greater(t, templateErrors, 0, "Should have template errors with empty context")
				assert.Greater(t, staticSuccess, 0, "Should have successful static files")
			},
		},
		{
			name:    "Success: Template all Terraform files",
			tplType: Terraform,
			context: map[string]any{
				"var": map[string]any{
					"project_id": "12345",
					"name":       "tf-cluster",
					"stage":      "staging",
				},
				"cluster": map[string]any{
					"terraform": map[string]any{
						"kubernetesType": "ske",
					},
				},
			},
			wantErr: false, // Changed to false with proper context
			validate: func(t *testing.T, results []TemplateResult) {
				assert.NotEmpty(t, results)
				for _, result := range results {
					assert.Contains(t, result.Path, "terraform")
					assert.False(t, strings.Contains(result.Path, "helm"), "Should not include helm files")
				}
			},
		},
		{
			name:    "Success: Template all Helm files",
			tplType: Helm,
			context: map[string]any{
				"cluster": map[string]any{
					"type":             "hub",
					"name":             "helm-cluster",
					"stage":            "production",
					"dnsName":          "helm.example.com",
					"ingressClassName": "traefik",
					"ssoOrg":           "myorg",
					"ssoTeam":          "myteam",
					"argocd": map[string]any{
						"repo": map[string]any{
							"https": map[string]any{
								"managed": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "managed-service-catalog/helm",
									"targetRevision": "main",
								},
								"customer": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "customer-service-catalog/helm",
									"targetRevision": "main",
								},
							},
						},
					},
					"services": fullServiceContext(),
				},
				"catalog": fullCatalogContext(),
			},
			wantErr: false, // Changed to false with proper context
			validate: func(t *testing.T, results []TemplateResult) {
				assert.NotEmpty(t, results)
				for _, result := range results {
					assert.Contains(t, result.Path, "helm")
					assert.False(t, strings.Contains(result.Path, "terraform"), "Should not include terraform files")
				}
			},
		},
		{
			name:    "Error: Invalid template type",
			tplType: TemplateType(99),
			context: map[string]any{},
			wantErr: true,
			validate: func(t *testing.T, results []TemplateResult) {
				assert.Empty(t, results)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestFS(t)
			defer cleanup()

			results, err := TemplateFiles(TemplateOptions{
				Type: tt.tplType,
				Data: tt.context,
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.validate != nil {
				tt.validate(t, results)
			}
		})
	}
}
