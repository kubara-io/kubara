package envmap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnvMapManager(t *testing.T) {
	tests := []struct {
		name      string
		filePath  string
		delim     string
		envPrefix string
	}{
		{
			name:      "Create a new envMap manager with dot delimiter",
			filePath:  "/tmp/envmap.env",
			delim:     ".",
			envPrefix: "KUBARA_",
		},
		{
			name:      "Create a new envMap manager with empty prefix",
			filePath:  "/tmp/envmap.env",
			delim:     ".",
			envPrefix: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewEnvMapManager(tt.filePath, tt.delim, tt.envPrefix)
			assert.NotNil(t, got)
			assert.NotNil(t, got.K)
			assert.Equal(t, tt.filePath, got.filepath)
			assert.NotNil(t, got.envMap)
			assert.Equal(t, tt.envPrefix, got.envPrefix)
		})
	}
}

func TestManager_GetFilepath(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
		want     string
	}{
		{
			name:     "Returns correct filepath",
			filepath: "/tmp/test.env",
			want:     "/tmp/test.env",
		},
		{
			name:     "Returns empty filepath",
			filepath: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				filepath: tt.filepath,
				envMap:   &EnvMap{},
			}
			assert.Equal(t, tt.want, m.GetFilepath())
		})
	}
}

func TestManager_GetConfig(t *testing.T) {
	t.Run("Returns the envMap config", func(t *testing.T) {
		expectedConfig := &EnvMap{
			ProjectName:  "test-project",
			ProjectStage: "dev",
		}
		m := &Manager{
			envMap: expectedConfig,
		}
		got := m.GetConfig()
		assert.Equal(t, expectedConfig, got)
	})
}

func TestManager_SetDefaults(t *testing.T) {
	t.Run("Sets defaults on the envMap", func(t *testing.T) {
		m := NewEnvMapManager("/tmp/test.env", ".", "")
		m.SetDefaults()

		// Verify that defaults were set
		assert.Equal(t, "<...>", m.envMap.ProjectName)
		assert.Equal(t, "<...>", m.envMap.ProjectStage)
		assert.Equal(t, "<...>", m.envMap.DomainName)
	})
}

func TestManager_Load(t *testing.T) {
	tempDir := t.TempDir()

	// Valid env file content
	validEnvContent := `PROJECT_NAME='test-project'
PROJECT_STAGE='dev'
DOCKERCONFIG_BASE64='dGVzdC1kb2NrZXItY29uZmlnCg=='
ARGOCD_WIZARD_ACCOUNT_PASSWORD='password123'
ARGOCD_HELM_REPO_USERNAME='helm-user'
ARGOCD_HELM_REPO_PASSWORD='helm-pass'
ARGOCD_HELM_REPO_URL='https://helm.example.com'
ARGOCD_GIT_HTTPS_URL='https://github.com/example/repo.git'
ARGOCD_GIT_PAT_OR_PASSWORD='github-token'
ARGOCD_GIT_USERNAME='git-user'
DOMAIN_NAME='example.com'
`

	validFilepath := filepath.Join(tempDir, "valid.env")
	require.NoError(t, os.WriteFile(validFilepath, []byte(validEnvContent), 0644))

	// Invalid env file content (malformed)
	invalidEnvContent := `PROJECT_NAME='unclosed quote
PROJECT_STAGE='dev'`
	invalidFilepath := filepath.Join(tempDir, "invalid.env")
	require.NoError(t, os.WriteFile(invalidFilepath, []byte(invalidEnvContent), 0644))

	tests := []struct {
		name      string
		filepath  string
		envVars   map[string]string
		wantErr   bool
		checkFunc func(*testing.T, *EnvMap)
	}{
		{
			name:     "Success: Loads valid env file",
			filepath: validFilepath,
			wantErr:  false,
			checkFunc: func(t *testing.T, em *EnvMap) {
				assert.Equal(t, "test-project", em.ProjectName)
				assert.Equal(t, "dev", em.ProjectStage)
				assert.Equal(t, "example.com", em.DomainName)
			},
		},
		{
			name:     "Success: Loads from non-existent file (uses env vars only)",
			filepath: filepath.Join(tempDir, "non-existent.env"),
			envVars: map[string]string{
				"PROJECT_NAME":  "env-project",
				"PROJECT_STAGE": "production",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, em *EnvMap) {
				assert.Equal(t, "env-project", em.ProjectName)
				assert.Equal(t, "production", em.ProjectStage)
			},
		},
		{
			name:     "Error: Invalid env file format",
			filepath: invalidFilepath,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for the test
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			m := NewEnvMapManager(tt.filepath, ".", "")
			err := m.Load()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, m.GetConfig())
				}
			}
		})
	}
}

func TestManager_Load_EnvironmentOverride(t *testing.T) {
	t.Run("Environment variables override file values", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a file with initial values
		fileContent := `PROJECT_NAME='file-project'
PROJECT_STAGE='dev'
DOMAIN_NAME='file-domain.com'`
		filepath := filepath.Join(tempDir, "test.env")
		require.NoError(t, os.WriteFile(filepath, []byte(fileContent), 0644))

		// Set environment variable that should override
		t.Setenv("PROJECT_NAME", "env-project")

		m := NewEnvMapManager(filepath, ".", "")
		err := m.Load()
		require.NoError(t, err)

		// Environment variable should override file value
		assert.Equal(t, "env-project", m.GetConfig().ProjectName)
		// File value should be used for non-overridden values
		assert.Equal(t, "dev", m.GetConfig().ProjectStage)
	})
}

func TestManager_Load_WithPrefix(t *testing.T) {
	t.Run("Loads environment variables with prefix", func(t *testing.T) {
		tempDir := t.TempDir()
		filepath := filepath.Join(tempDir, "test.env")

		// Set environment variables with prefix
		t.Setenv("KUBARA_PROJECT_NAME", "prefixed-project")
		t.Setenv("KUBARA_PROJECT_STAGE", "staging")

		m := NewEnvMapManager(filepath, ".", "KUBARA_")
		err := m.Load()
		require.NoError(t, err)

		assert.Equal(t, "prefixed-project", m.GetConfig().ProjectName)
		assert.Equal(t, "staging", m.GetConfig().ProjectStage)
	})
}

func TestManager_Validate(t *testing.T) {
	tests := []struct {
		name    string
		envMap  *EnvMap
		wantErr bool
	}{
		{
			name: "Valid config passes validation",
			envMap: &EnvMap{
				ProjectName:                 "test-project",
				ProjectStage:                "dev",
				DockerconfigBase64:          "dGVzdA==",
				ArgocdWizardAccountPassword: "pass",
				ArgocdHelmRepoUsername:      "user",
				ArgocdHelmRepoPassword:      "pass",
				ArgocdHelmRepoUrl:           "url",
				ArgocdGitHttpsUrl:           "url",
				ArgocdGitPatOrPassword:      "token",
				ArgocdGitUsername:           "user",
				DomainName:                  "example.com",
			},
			wantErr: false,
		},
		{
			name: "Invalid config fails validation",
			envMap: &EnvMap{
				ProjectName: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				envMap: tt.envMap,
			}
			err := m.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_ValidateAll(t *testing.T) {
	tests := []struct {
		name    string
		envMap  *EnvMap
		wantErr bool
	}{
		{
			name: "Valid config passes ValidateAll",
			envMap: &EnvMap{
				ProjectName:                 "test-project",
				ProjectStage:                "dev",
				DockerconfigBase64:          "dGVzdA==",
				ArgocdWizardAccountPassword: "pass",
				ArgocdHelmRepoUsername:      "user",
				ArgocdHelmRepoPassword:      "pass",
				ArgocdHelmRepoUrl:           "url",
				ArgocdGitHttpsUrl:           "url",
				ArgocdGitPatOrPassword:      "token",
				ArgocdGitUsername:           "user",
				DomainName:                  "example.com",
			},
			wantErr: false,
		},
		{
			name: "Invalid config fails ValidateAll",
			envMap: &EnvMap{
				ProjectName: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				envMap: tt.envMap,
			}
			err := m.ValidateAll()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_SaveToFile(t *testing.T) {
	tempDir := t.TempDir()

	testEnvMap := &EnvMap{
		ProjectName:                 "test-project",
		ProjectStage:                "dev",
		DockerconfigBase64:          "dGVzdC1kb2NrZXItY29uZmlnCg==",
		ArgocdWizardAccountPassword: "password123",
		ArgocdHelmRepoUsername:      "helm-user",
		ArgocdHelmRepoPassword:      "helm-pass",
		ArgocdHelmRepoUrl:           "https://helm.example.com",
		ArgocdGitHttpsUrl:           "https://github.com/example/repo.git",
		ArgocdGitPatOrPassword:      "github-token",
		ArgocdGitUsername:           "git-user",
		DomainName:                  "example.com",
	}

	successFilepath := filepath.Join(tempDir, "output.env")

	// Setup for permission error case
	readOnlyDir := filepath.Join(tempDir, "readonly_dir")
	require.NoError(t, os.Mkdir(readOnlyDir, 0755))
	require.NoError(t, os.Chmod(readOnlyDir, 0555))
	permissionErrorFilepath := filepath.Join(readOnlyDir, "output.env")

	tests := []struct {
		name      string
		filepath  string
		envMap    *EnvMap
		wantErr   assert.ErrorAssertionFunc
		postCheck func(*testing.T, string)
	}{
		{
			name:     "Success: Saves envMap to file",
			filepath: successFilepath,
			envMap:   testEnvMap,
			wantErr:  assert.NoError,
			postCheck: func(t *testing.T, filepath string) {
				assert.FileExists(t, filepath)

				content, err := os.ReadFile(filepath)
				require.NoError(t, err)

				contentStr := string(content)
				assert.Contains(t, contentStr, "PROJECT_NAME='test-project'")
				assert.Contains(t, contentStr, "PROJECT_STAGE='dev'")
				assert.Contains(t, contentStr, "DOMAIN_NAME='example.com'")

				// Verify file permissions
				info, err := os.Stat(filepath)
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
			},
		},
		{
			name:     "Error: Fails when saving to read-only directory",
			filepath: permissionErrorFilepath,
			envMap:   testEnvMap,
			wantErr:  assert.Error,
			postCheck: func(t *testing.T, filepath string) {
				assert.NoFileExists(t, filepath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				filepath: tt.filepath,
				envMap:   tt.envMap,
			}

			err := m.SaveToFile(tt.filepath)
			tt.wantErr(t, err)

			if tt.postCheck != nil {
				tt.postCheck(t, tt.filepath)
			}
		})
	}
}

func TestManager_ValidateAndSaveToFile(t *testing.T) {
	tempDir := t.TempDir()

	validEnvMap := &EnvMap{
		ProjectName:                 "test-project",
		ProjectStage:                "dev",
		DockerconfigBase64:          "dGVzdA==",
		ArgocdWizardAccountPassword: "pass",
		ArgocdHelmRepoUsername:      "user",
		ArgocdHelmRepoPassword:      "pass",
		ArgocdHelmRepoUrl:           "url",
		ArgocdGitHttpsUrl:           "url",
		ArgocdGitPatOrPassword:      "token",
		ArgocdGitUsername:           "user",
		DomainName:                  "example.com",
	}

	invalidEnvMap := &EnvMap{
		ProjectName: "",
	}

	tests := []struct {
		name      string
		filepath  string
		envMap    *EnvMap
		wantErr   bool
		postCheck func(*testing.T, string)
	}{
		{
			name:     "Success: Validates and saves valid config",
			filepath: filepath.Join(tempDir, "valid_output.env"),
			envMap:   validEnvMap,
			wantErr:  false,
			postCheck: func(t *testing.T, filepath string) {
				assert.FileExists(t, filepath)
			},
		},
		{
			name:     "Error: Fails validation and does not save",
			filepath: filepath.Join(tempDir, "invalid_output.env"),
			envMap:   invalidEnvMap,
			wantErr:  true,
			postCheck: func(t *testing.T, filepath string) {
				assert.NoFileExists(t, filepath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				filepath: tt.filepath,
				envMap:   tt.envMap,
			}

			err := m.ValidateAndSaveToFile(tt.filepath)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.postCheck != nil {
				tt.postCheck(t, tt.filepath)
			}
		})
	}
}

func TestManager_GenerateEnvExample(t *testing.T) {
	tests := []struct {
		name      string
		envMap    *EnvMap
		checkFunc func(*testing.T, []byte)
	}{
		{
			name:   "Generates env example with defaults",
			envMap: &EnvMap{},
			checkFunc: func(t *testing.T, output []byte) {
				outputStr := string(output)

				// Check that documentation comments are included
				assert.Contains(t, outputStr, "# Step 1: Init and Terraform")
				assert.Contains(t, outputStr, "### Project related values")

				// Check that all required fields are present with default values
				assert.Contains(t, outputStr, "PROJECT_NAME='<...>'")
				assert.Contains(t, outputStr, "PROJECT_STAGE='<...>'")
				assert.Contains(t, outputStr, "DOMAIN_NAME='<...>'")
				assert.Contains(t, outputStr, "DOCKERCONFIG_BASE64='<...>'")
				assert.Contains(t, outputStr, "ARGOCD_WIZARD_ACCOUNT_PASSWORD='<...>'")
			},
		},
		{
			name: "Generates env example with existing values",
			envMap: &EnvMap{
				ProjectName:  "existing-project",
				ProjectStage: "production",
			},
			checkFunc: func(t *testing.T, output []byte) {
				outputStr := string(output)

				// Should still use default values from tags, not existing values
				assert.Contains(t, outputStr, "PROJECT_NAME='<...>'")
				assert.Contains(t, outputStr, "PROJECT_STAGE='<...>'")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{
				envMap: tt.envMap,
			}

			output, err := m.GenerateEnvExample()
			require.NoError(t, err)
			require.NotEmpty(t, output)

			if tt.checkFunc != nil {
				tt.checkFunc(t, output)
			}
		})
	}
}

func TestManager_GenerateEnvExample_Format(t *testing.T) {
	t.Run("Generated env example has proper format", func(t *testing.T) {
		m := NewEnvMapManager("/tmp/test.env", ".", "")

		output, err := m.GenerateEnvExample()
		require.NoError(t, err)

		lines := strings.Split(string(output), "\n")

		// Verify the output has multiple lines
		assert.Greater(t, len(lines), 10)

		// Verify that comment lines start with #
		commentCount := 0
		for _, line := range lines {
			if strings.HasPrefix(line, "#") {
				commentCount++
			}
		}
		assert.Greater(t, commentCount, 0, "Should have comment lines")

		// Verify that variable lines have the format KEY='VALUE'
		varCount := 0
		for _, line := range lines {
			if strings.Contains(line, "=") && !strings.HasPrefix(line, "#") {
				assert.Contains(t, line, "='")
				varCount++
			}
		}
		assert.Greater(t, varCount, 0, "Should have variable lines")
	})
}

func TestManager_SaveToFile_DoesNotIncludeDocFields(t *testing.T) {
	t.Run("SaveToFile only saves fields with koanf tags", func(t *testing.T) {
		tempDir := t.TempDir()
		filepath := filepath.Join(tempDir, "output.env")

		m := &Manager{
			filepath: filepath,
			envMap: &EnvMap{
				ProjectName:  "test",
				ProjectStage: "dev",
			},
		}

		err := m.SaveToFile(filepath)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath)
		require.NoError(t, err)

		contentStr := string(content)

		// Should contain actual fields
		assert.Contains(t, contentStr, "PROJECT_NAME='test'")
		assert.Contains(t, contentStr, "PROJECT_STAGE='dev'")

		// Should NOT contain documentation comments
		assert.NotContains(t, contentStr, "# Step 1")
		assert.NotContains(t, contentStr, "###")
	})
}
