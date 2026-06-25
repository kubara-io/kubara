package render

import (
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
		"velero": map[string]any{
			"status": "disabled",
			"config": map[string]any{
				"backupMode":    "fs-backup",
				"backupStorage": map[string]any{"create": true, "region": "eu01"},
			},
		},
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
			"velero":                  map[string]any{"chartPath": "velero"},
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
			name:  "does not strip providers/<name> outside terraform context",
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
	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "t-cloud-public",
		Data: map[string]any{
			"cluster": map[string]any{
				"name":    "test-cluster",
				"stage":   "dev",
				"dnsName": "test.example.com",
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

	paths := make([]string, 0, len(results))
	for _, result := range results {
		require.NoError(t, result.Error)
		paths = append(paths, result.Path)
	}

	// Assert on generator behavior (which files are selected for the provider/type),
	// not on rendered Terraform template content — see AGENTS.md "Catalog Boundary".
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/cce-cluster/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/network/main.tf")
	assert.Contains(t, paths, "managed-service-catalog/terraform/providers/t-cloud-public/modules/storage-classes/main.tf")
	assert.Contains(t, paths, "customer-service-catalog/terraform/providers/t-cloud-public/example/infrastructure/main.tf.tplt")
	assert.Contains(t, paths, "customer-service-catalog/terraform/providers/t-cloud-public/example/openbao/main.tf.tplt")
	assert.Contains(t, paths, "customer-service-catalog/terraform/providers/t-cloud-public/example/openbao/secrets.tf-example.tplt")
	assert.NotContains(t, paths, "managed-service-catalog/terraform/providers/stackit/modules/ske-cluster/main.tf")
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
