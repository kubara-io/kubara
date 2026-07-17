package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	internaltestutil "github.com/kubara-io/kubara/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testBootstrapCatalogPath string
	testGeneralCatalogPath   string
)

func init() {
	root, err := os.MkdirTemp("", "kubara-render-catalog-tests-*")
	if err != nil {
		panic(err)
	}

	bootstrapPath, generalPath, err := internaltestutil.CreateCatalogFixtures(filepath.Join(root, "catalogs"))
	if err != nil {
		panic(err)
	}

	testBootstrapCatalogPath = bootstrapPath
	testGeneralCatalogPath = generalPath
}

func testTemplateCatalogs() []string {
	return []string{
		testBootstrapCatalogPath,
		testGeneralCatalogPath,
	}
}

func fullServiceContext() map[string]any {
	return map[string]any{
		"cert-manager": map[string]any{"status": "enabled", "config": map[string]any{"clusterIssuer": map[string]any{"name": "letsencrypt-prod", "email": "admin@example.com", "server": "https://acme-staging-v02.api.letsencrypt.org/directory"}}},
	}
}

func fullCatalogContext() map[string]any {
	return map[string]any{
		"services": map[string]any{
			"argo-cd":        map[string]any{"chartPath": "argo-cd"},
			"bootstrap-crds": map[string]any{"chartPath": "bootstrap-crds"},
			"cert-manager":   map[string]any{"chartPath": "cert-manager"},
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
			name:  "strips stackit under terraform",
			input: "platform-configs/terraform/stackit/example/infrastructure/main.tf.tplt",
			want:  "platform-configs/terraform/example/infrastructure/main.tf.tplt",
		},
		{
			name:  "leaves platform-components terraform path unchanged",
			input: "platform-components/terraform/stackit/modules/ske-cluster/main.tf",
			want:  "platform-components/terraform/stackit/modules/ske-cluster/main.tf",
		},
		{
			name:  "leaves non-provider terraform path unchanged",
			input: "platform-components/terraform/images/public-cloud-0.png",
			want:  "platform-components/terraform/images/public-cloud-0.png",
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
		Catalogs: testTemplateCatalogs(),
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
	var cceClusterModule string
	var infrastructureMain string
	for _, result := range results {
		require.NoError(t, result.Error)
		paths = append(paths, result.Path)
		switch result.Path {
		case "platform-components/terraform/t-cloud-public/modules/cce-cluster/main.tf":
			cceClusterModule = result.Content
		case "platform-configs/terraform/t-cloud-public/infrastructure/main.tf.tplt":
			infrastructureMain = result.Content
		}
	}

	assert.Contains(t, paths, "platform-components/terraform/t-cloud-public/modules/cce-cluster/main.tf")
	assert.Contains(t, paths, "platform-components/terraform/t-cloud-public/modules/network/main.tf")
	assert.Contains(t, paths, "platform-components/terraform/t-cloud-public/modules/storage-classes/main.tf")
	assert.Contains(t, paths, "platform-configs/terraform/t-cloud-public/infrastructure/main.tf.tplt")
	assert.NotContains(t, paths, "platform-components/terraform/stackit/modules/ske-cluster/main.tf")
	require.NotEmpty(t, cceClusterModule)
	assert.Contains(t, cceClusterModule, `file_permission = "0600"`)
	require.NotEmpty(t, infrastructureMain)
	assert.Contains(t, infrastructureMain, `source = "../../../../platform-components/terraform/t-cloud-public/modules/cce-cluster"`)
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
								"components": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "platform-components/helm",
									"targetRevision": "main",
								},
								"configs": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "platform-configs",
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
					"services": fullServiceContext(),
				},
				"catalog": fullCatalogContext(),
			},
			wantErr: false, // Changed to false with proper context
			validate: func(t *testing.T, results []TemplateResult) {
				assert.NotEmpty(t, results)
				for _, result := range results {
					assert.True(t, strings.Contains(result.Path, "terraform"), "Path %q should contain 'terraform'", result.Path)
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
								"components": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "platform-components/helm",
									"targetRevision": "main",
								},
								"configs": map[string]any{
									"url":            "https://github.com/example/repo",
									"path":           "platform-configs",
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
			provider := ""
			if tt.tplType == Terraform || tt.tplType == All {
				provider = "stackit"
			}

			results, err := TemplateFiles(TemplateOptions{
				Type:     tt.tplType,
				Provider: provider,
				Catalogs: testTemplateCatalogs(),
				Data:     tt.context,
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
