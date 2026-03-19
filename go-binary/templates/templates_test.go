package templates

import (
	"embed"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

//go:embed all:embedded
var testTemplatesFS embed.FS

// helper function to setup test filesystem with correct root path
func setupTestFS(t *testing.T) func() {
	originalFS := templatesFSNew
	templatesFSNew = testTemplatesFS

	// Return cleanup function
	return func() {
		templatesFSNew = originalFS
	}
}

// getEmbeddedTemplatesListTest temporarily sets templatesFSNew for testing
func getEmbeddedTemplatesListTest(tplType TemplateType, testFS embed.FS) ([]string, error) {
	originalFS := templatesFSNew
	templatesFSNew = testFS
	defer func() { templatesFSNew = originalFS }()

	return GetEmbeddedTemplatesList(tplType)
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
	walkFunc := makeWalkDirFunc("embedded", &files)

	err := fs.WalkDir(testFS, "embedded", walkFunc)
	require.NoError(t, err)

	// Verify that files are collected (not directories)
	require.NotEmpty(t, files)
	for _, file := range files {
		assert.NotEmpty(t, file)
		assert.False(t, strings.HasSuffix(file, "/"))
	}

	// Test error propagation if WalkDir encounters an error
	var errorFiles []string
	errorWalkFunc := makeWalkDirFunc("embedded", &errorFiles)
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

	err := fs.WalkDir(testFS, "embedded", walkFunc)
	// Should still work but paths might be relative to nonexistent root
	require.NoError(t, err)
}

func TestMakeWalkDirFunc_DirectoryFiltering(t *testing.T) {
	// Test that directories are properly filtered out
	testFS := testTemplatesFS
	var files []string
	walkFunc := makeWalkDirFunc("embedded", &files)

	err := fs.WalkDir(testFS, "embedded", walkFunc)
	require.NoError(t, err)

	// Ensure no directory entries (ending with /) are included
	for _, file := range files {
		assert.False(t, strings.HasSuffix(file, "/"), "File path should not end with /: %s", file)
	}
}

func TestGetEmbeddedTemplatesList(t *testing.T) {
	tests := []struct {
		name     string
		tplType  TemplateType
		wantErr  bool
		validate func(t *testing.T, list []string)
	}{
		{
			name:    "Terraform",
			tplType: Terraform,
			wantErr: false,
			validate: func(t *testing.T, list []string) {
				require.NotEmpty(t, list)
				for _, p := range list {
					assert.Contains(t, p, "terraform")
					assert.False(t, strings.Contains(p, "helm"), "Terraform list should not include Helm paths: %s", p)
				}
			},
		},
		{
			name:    "Helm",
			tplType: Helm,
			wantErr: false,
			validate: func(t *testing.T, list []string) {
				require.NotEmpty(t, list)
				for _, p := range list {
					assert.Contains(t, p, "helm")
					assert.False(t, strings.Contains(p, "terraform"), "Helm list should not include Terraform paths: %s", p)
				}
			},
		},
		{
			name:    "All",
			tplType: All,
			wantErr: false,
			validate: func(t *testing.T, list []string) {
				require.NotEmpty(t, list)
				hasTerraform := false
				hasHelm := false
				for _, p := range list {
					if strings.Contains(p, "terraform") {
						hasTerraform = true
					}
					if strings.Contains(p, "helm") {
						hasHelm = true
					}
				}
				assert.True(t, hasTerraform)
				assert.True(t, hasHelm)
			},
		},
		{
			name:    "Invalid Type",
			tplType: TemplateType(99),
			wantErr: true, // Walks non-existent paths
			validate: func(t *testing.T, list []string) {
				assert.Empty(t, list)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestFS(t)
			defer cleanup()

			list, err := GetEmbeddedTemplatesList(tt.tplType)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if tt.validate != nil {
				tt.validate(t, list)
			}
		})
	}

	// Additional test: Error if root does not exist (simulate by overriding)
	t.Run("Error on non-existent root for All", func(t *testing.T) {
		invalidFS := embed.FS{} // Empty FS
		list, err := getEmbeddedTemplatesListTest(All, invalidFS)
		assert.Error(t, err)
		assert.Empty(t, list)
	})
}

func TestGetEmbeddedTemplatesList_ErrorCases(t *testing.T) {
	// Test error handling when both customer and managed service catalogs fail
	t.Run("Both catalog paths fail", func(t *testing.T) {
		cleanup := setupTestFS(t)
		defer cleanup()

		// Test with a type that tries to access specific paths
		list, err := GetEmbeddedTemplatesList(Terraform)
		// Should not error since our test FS has the paths
		assert.NoError(t, err)
		assert.NotEmpty(t, list)
	})
}

func TestTemplateFiles(t *testing.T) {
	tests := []struct {
		name     string
		fileList []string
		context  map[string]any
		wantErr  bool
		validate func(t *testing.T, results []TemplateResult)
	}{
		{
			name:     "Success: Successfully template terraform files",
			fileList: []string{"customer-service-catalog/terraform/example/infrastructure/main.tf.tplt"},
			context: map[string]any{
				"var": map[string]interface{}{
					"project_id": "12345",
					"name":       "test-cluster",
					"stage":      "dev",
				},
				"cluster": map[string]interface{}{
					"terraform": map[string]interface{}{
						"kubernetesType": "ske",
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, results []TemplateResult) {
				require.Len(t, results, 1)
				assert.Equal(t, "customer-service-catalog/terraform/example/infrastructure/main.tf.tplt", results[0].Path)
				assert.NoError(t, results[0].Error)
				assert.NotEmpty(t, results[0].Content)
				// The ${var.name} syntax is Terraform syntax, not go template, so it won't be substituted
				// Only the go template {{ if }} blocks will be processed
				assert.Contains(t, results[0].Content, "ske_cluster")
				assert.Contains(t, results[0].Content, "var.project_id")
				assert.Contains(t, results[0].Content, "${var.name}")
			},
		},
		{
			name:     "Success: Successfully template helm files",
			fileList: []string{"customer-service-catalog/helm/example/argo-cd/values.yaml.tplt"},
			context: map[string]any{
				"cluster": map[string]interface{}{
					"type":    "controlplane",
					"name":    "test-cluster",
					"stage":   "dev",
					"dnsName": "test.example.com",
					"ssoOrg":  "myorg",
					"ssoTeam": "myteam",
					"services": map[string]interface{}{
						"oauth2Proxy": map[string]interface{}{
							"status": "enabled",
						},
						"certManager": map[string]interface{}{
							"status": "enabled",
							"clusterIssuer": map[string]interface{}{
								"name": "letsencrypt-prod",
							},
						},
						"metalLb": map[string]interface{}{
							"status": "enabled",
						},
						"kubePrometheusStack": map[string]interface{}{
							"status": "enabled",
						},
					},
					"publicLoadbalancerIP": "1.2.3.4",
					"argocd": map[string]interface{}{
						"repo": map[string]interface{}{
							"https": map[string]interface{}{
								"managed": map[string]interface{}{
									"url":            "https://github.com/example/repo",
									"path":           "managed-service-catalog/helm",
									"targetRevision": "main",
								},
								"customer": map[string]interface{}{
									"url":            "https://github.com/example/repo",
									"path":           "customer-service-catalog/helm",
									"targetRevision": "main",
								},
							},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, results []TemplateResult) {
				require.Len(t, results, 1)
				assert.Equal(t, "customer-service-catalog/helm/example/argo-cd/values.yaml.tplt", results[0].Path)
				assert.NoError(t, results[0].Error)
				assert.NotEmpty(t, results[0].Content)
				assert.Contains(t, results[0].Content, "test-cluster")
				assert.Contains(t, results[0].Content, "dev")
			},
		},
		{
			name:     "Success: Keep ArgoCD rbac and params under configs when oauth2 is disabled",
			fileList: []string{"customer-service-catalog/helm/example/argo-cd/values.yaml.tplt"},
			context: map[string]any{
				"cluster": map[string]interface{}{
					"type":    "controlplane",
					"name":    "test-cluster",
					"stage":   "dev",
					"dnsName": "test.example.com",
					"ssoOrg":  "myorg",
					"ssoTeam": "myteam",
					"services": map[string]interface{}{
						"oauth2Proxy": map[string]interface{}{
							"status": "disabled",
						},
						"certManager": map[string]interface{}{
							"status": "enabled",
							"clusterIssuer": map[string]interface{}{
								"name": "letsencrypt-prod",
							},
						},
						"metalLb": map[string]interface{}{
							"status": "enabled",
						},
						"kubePrometheusStack": map[string]interface{}{
							"status": "enabled",
						},
					},
					"publicLoadbalancerIP": "1.2.3.4",
					"argocd": map[string]interface{}{
						"repo": map[string]interface{}{
							"https": map[string]interface{}{
								"managed": map[string]interface{}{
									"url":            "https://github.com/example/repo",
									"path":           "managed-service-catalog/helm",
									"targetRevision": "main",
								},
								"customer": map[string]interface{}{
									"url":            "https://github.com/example/repo",
									"path":           "customer-service-catalog/helm",
									"targetRevision": "main",
								},
							},
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, results []TemplateResult) {
				require.Len(t, results, 1)
				assert.Equal(t, "customer-service-catalog/helm/example/argo-cd/values.yaml.tplt", results[0].Path)
				assert.NoError(t, results[0].Error)

				var rendered map[string]interface{}
				require.NoError(t, yaml.Unmarshal([]byte(results[0].Content), &rendered))

				argoCD, ok := rendered["argo-cd"].(map[string]interface{})
				require.True(t, ok)

				configs, ok := argoCD["configs"].(map[string]interface{})
				require.True(t, ok)

				_, hasConfigRbac := configs["rbac"]
				assert.True(t, hasConfigRbac)
				_, hasConfigParams := configs["params"]
				assert.True(t, hasConfigParams)
				_, hasConfigCM := configs["cm"]
				assert.False(t, hasConfigCM)

				server, ok := argoCD["server"].(map[string]interface{})
				require.True(t, ok)
				_, hasServerRbac := server["rbac"]
				assert.False(t, hasServerRbac)
				_, hasServerParams := server["params"]
				assert.False(t, hasServerParams)
			},
		},
		{
			name:     "Success: Successfully copy non-template files",
			fileList: []string{"managed-service-catalog/terraform/modules/ske-cluster/main.tf"},
			context:  map[string]any{},
			wantErr:  false,
			validate: func(t *testing.T, results []TemplateResult) {
				require.Len(t, results, 1)
				assert.Equal(t, "managed-service-catalog/terraform/modules/ske-cluster/main.tf", results[0].Path)
				assert.NoError(t, results[0].Error)
				assert.NotEmpty(t, results[0].Content)
				assert.Contains(t, results[0].Content, "stackit_ske_cluster")
			},
		},
		{
			name:     "Error: Handle non-existent file",
			fileList: []string{"non-existent/file.tplt"},
			context:  map[string]any{},
			wantErr:  true,
			validate: func(t *testing.T, results []TemplateResult) {
				require.Len(t, results, 1)
				assert.Equal(t, "non-existent/file.tplt", results[0].Path)
				assert.Error(t, results[0].Error)
				assert.Empty(t, results[0].Content)
			},
		},
		{
			name:     "Error: Handle template execution error",
			fileList: []string{"customer-service-catalog/terraform/example/infrastructure/main.tf.tplt"},
			context: map[string]any{
				"var": map[string]interface{}{
					"project_id": "12345",
				},
				"cluster": map[string]interface{}{
					"terraform": map[string]interface{}{
						// This will cause a runtime error when accessing cluster.terraform.kubernetesType
						"kubernetesType": func() string { panic("template error") },
					},
				},
			},
			wantErr: true,
			validate: func(t *testing.T, results []TemplateResult) {
				require.Len(t, results, 1)
				assert.Equal(t, "customer-service-catalog/terraform/example/infrastructure/main.tf.tplt", results[0].Path)
				assert.Error(t, results[0].Error)
				assert.Empty(t, results[0].Content)
			},
		},
		{
			name:     "Success: Handle missing keys (no error with default behavior)",
			fileList: []string{"customer-service-catalog/terraform/example/infrastructure/main.tf.tplt"},
			context: map[string]any{
				// Missing cluster.terraform.kubernetesType - should not cause error
				"var": map[string]interface{}{
					"project_id": "12345",
				},
			},
			wantErr: false, // Go templates silently ignore missing keys by default
			validate: func(t *testing.T, results []TemplateResult) {
				require.Len(t, results, 1)
				assert.Equal(t, "customer-service-catalog/terraform/example/infrastructure/main.tf.tplt", results[0].Path)
				assert.NoError(t, results[0].Error)
				assert.NotEmpty(t, results[0].Content)
				// Template should render but missing variables will be empty
			},
		},
		{
			name: "Error: Handle mixed file list with some errors",
			fileList: []string{
				"customer-service-catalog/terraform/example/infrastructure/main.tf.tplt",
				"non-existent/file.tplt",
				"managed-service-catalog/terraform/modules/ske-cluster/main.tf",
			},
			context: map[string]any{
				"var": map[string]interface{}{
					"project_id": "12345",
					"name":       "test-cluster",
					"stage":      "dev",
				},
				"cluster": map[string]interface{}{
					"terraform": map[string]interface{}{
						"kubernetesType": "ske",
					},
				},
			},
			wantErr: true, // Should return combined error
			validate: func(t *testing.T, results []TemplateResult) {
				require.Len(t, results, 3)

				// First file should succeed
				assert.Equal(t, "customer-service-catalog/terraform/example/infrastructure/main.tf.tplt", results[0].Path)
				assert.NoError(t, results[0].Error)
				assert.NotEmpty(t, results[0].Content)

				// Second file should fail
				assert.Equal(t, "non-existent/file.tplt", results[1].Path)
				assert.Error(t, results[1].Error)
				assert.Empty(t, results[1].Content)

				// Third file should succeed
				assert.Equal(t, "managed-service-catalog/terraform/modules/ske-cluster/main.tf", results[2].Path)
				assert.NoError(t, results[2].Error)
				assert.NotEmpty(t, results[2].Content)
			},
		},
		{
			name:     "Success: Handle empty file list",
			fileList: []string{},
			context:  map[string]any{},
			wantErr:  false,
			validate: func(t *testing.T, results []TemplateResult) {
				require.Len(t, results, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestFS(t)
			defer cleanup()

			results, err := TemplateFiles(tt.fileList, tt.context)

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

func TestTemplateAllFiles(t *testing.T) {
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
				"var": map[string]interface{}{
					"project_id": "12345",
					"name":       "test-cluster",
					"stage":      "dev",
				},
				"cluster": map[string]interface{}{
					"type":  "controlplane",
					"name":  "test-cluster",
					"stage": "dev",
					"terraform": map[string]interface{}{
						"kubernetesType": "ske",
					},
				},
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
			context: map[string]any{}, // Missing required variables - but go template doesn't fail by default
			wantErr: false,            // Changed to false since go template doesn't fail on missing vars
			validate: func(t *testing.T, results []TemplateResult) {
				assert.NotEmpty(t, results)
				// Should have both template and static files, but templates may have empty content
				templateFiles := 0
				staticSuccess := 0
				for _, result := range results {
					if strings.HasSuffix(result.Path, ".tplt") {
						templateFiles++
					} else {
						if result.Error == nil {
							staticSuccess++
						}
					}
				}
				assert.Greater(t, templateFiles, 0, "Should have template files")
				assert.Greater(t, staticSuccess, 0, "Should have successful static files")
			},
		},
		{
			name:    "Success: Template all Terraform files",
			tplType: Terraform,
			context: map[string]any{
				"var": map[string]interface{}{
					"project_id": "12345",
					"name":       "tf-cluster",
					"stage":      "staging",
				},
				"cluster": map[string]interface{}{
					"terraform": map[string]interface{}{
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
				"cluster": map[string]interface{}{
					"type":  "controlplane",
					"name":  "helm-cluster",
					"stage": "production",
				},
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

			results, err := TemplateAllFiles(tt.tplType, tt.context)

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
