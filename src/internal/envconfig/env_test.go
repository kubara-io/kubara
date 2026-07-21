package envconfig

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorEnvMap_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *ErrorEnvMap
		wantMsg string
	}{
		{
			name: "Error message is formatted correctly",
			err: &ErrorEnvMap{
				Message: "test error message",
				Err:     ErrEnvsNotSet,
			},
			wantMsg: "Error: test error message",
		},
		{
			name: "Empty message is handled",
			err: &ErrorEnvMap{
				Message: "",
				Err:     ErrDefaultIsSet,
			},
			wantMsg: "Error: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			assert.Equal(t, tt.wantMsg, got)
		})
	}
}

func TestErrorEnvMap_Unwrap(t *testing.T) {
	tests := []struct {
		name string
		err  *ErrorEnvMap
		want error
	}{
		{
			name: "Unwraps ErrEnvsNotSet correctly",
			err: &ErrorEnvMap{
				Message: "test message",
				Err:     ErrEnvsNotSet,
			},
			want: ErrEnvsNotSet,
		},
		{
			name: "Unwraps ErrDefaultIsSet correctly",
			err: &ErrorEnvMap{
				Message: "test message",
				Err:     ErrDefaultIsSet,
			},
			want: ErrDefaultIsSet,
		},
		{
			name: "Unwraps nil error",
			err: &ErrorEnvMap{
				Message: "test message",
				Err:     nil,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Unwrap()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEnvMap_Validate(t *testing.T) {
	// Helper to create a valid EnvMap with all required fields set
	validEnvMap := func() *EnvMap {
		return &EnvMap{
			ProjectName:                 "test-project",
			ProjectStage:                "dev",
			ArgocdWizardAccountPassword: "password123",
			ArgocdHelmRepoUsername:      "helm-user",
			ArgocdHelmRepoPassword:      "helm-pass",
			ArgocdHelmRepoUrl:           "https://helm.example.com",
			ArgocdGitHttpsUrl:           "https://github.com/example/repo.git",
		}
	}

	tests := []struct {
		name    string
		envMap  *EnvMap
		wantErr bool
		errType error
	}{
		{
			name:    "Valid EnvMap passes validation",
			envMap:  validEnvMap(),
			wantErr: false,
		},
		{
			name: "Missing required field fails validation",
			envMap: func() *EnvMap {
				em := validEnvMap()
				em.ProjectName = ""
				return em
			}(),
			wantErr: true,
			errType: ErrEnvsNotSet,
		},
		{
			name: "Field with default value fails validation",
			envMap: func() *EnvMap {
				em := validEnvMap()
				em.ProjectName = "<...>"
				return em
			}(),
			wantErr: true,
			errType: ErrDefaultIsSet,
		},
		{
			name: "Missing optional fields pass validation",
			envMap: func() *EnvMap {
				em := validEnvMap()
				em.DockerconfigBase64 = ""
				em.ArgocdHelmRepoUsername = ""
				em.ArgocdHelmRepoPassword = ""
				em.ArgocdHelmRepoUrl = ""
				em.ArgocdGitPatOrPassword = ""
				em.ArgocdGitUsername = ""
				return em
			}(),
			wantErr: false,
		},
		{
			name: "Multiple missing required fields",
			envMap: func() *EnvMap {
				em := validEnvMap()
				em.ProjectName = ""
				em.ProjectStage = ""
				return em
			}(),
			wantErr: true,
			errType: ErrEnvsNotSet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.envMap.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.True(t, errors.Is(err, tt.errType), "Expected error type %v, got %v", tt.errType, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEnvMap_setDefaults(t *testing.T) {
	tests := []struct {
		name          string
		envMap        *EnvMap
		checkField    func(*EnvMap) string
		expectedValue string
		fieldName     string
	}{
		{
			name:   "Sets default for ProjectName when empty",
			envMap: &EnvMap{},
			checkField: func(em *EnvMap) string {
				return em.ProjectName
			},
			expectedValue: "<...>",
			fieldName:     "ProjectName",
		},
		{
			name: "Does not overwrite existing value",
			envMap: &EnvMap{
				ProjectName: "existing-project",
			},
			checkField: func(em *EnvMap) string {
				return em.ProjectName
			},
			expectedValue: "existing-project",
			fieldName:     "ProjectName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.envMap.setDefaults()
			got := tt.checkField(tt.envMap)
			assert.Equal(t, tt.expectedValue, got, "Field %s", tt.fieldName)
		})
	}
}

func TestEnvMap_setDefaults_AllFields(t *testing.T) {
	t.Run("All empty fields get defaults", func(t *testing.T) {
		em := &EnvMap{}
		em.setDefaults()

		// Verify that all fields with default tags are set
		assert.Equal(t, "<...>", em.ProjectName)
		assert.Equal(t, "<...>", em.ProjectStage)
		assert.Equal(t, "", em.DockerconfigBase64)
		assert.Equal(t, "<...>", em.ArgocdWizardAccountPassword)
		assert.Equal(t, "", em.ArgocdHelmRepoUsername)
		assert.Equal(t, "", em.ArgocdHelmRepoPassword)
		assert.Equal(t, "", em.ArgocdHelmRepoUrl)
		assert.Equal(t, "https", em.ArgocdGitAuthMode)
		assert.Equal(t, "", em.ArgocdGitUrl)
		assert.Equal(t, "", em.ArgocdGitHttpsUrl)
		assert.Equal(t, "", em.ArgocdGitPatOrPassword)
		assert.Equal(t, "", em.ArgocdGitUsername)
	})
}

func TestErrorEnvMap_ErrorWrapping(t *testing.T) {
	t.Run("errors.Is works with wrapped errors", func(t *testing.T) {
		err := &ErrorEnvMap{
			Message: "test error",
			Err:     ErrEnvsNotSet,
		}

		assert.True(t, errors.Is(err, ErrEnvsNotSet))
		assert.False(t, errors.Is(err, ErrDefaultIsSet))
	})

	t.Run("errors.As works with ErrorEnvMap", func(t *testing.T) {
		err := &ErrorEnvMap{
			Message: "test error",
			Err:     ErrEnvsNotSet,
		}

		var target *ErrorEnvMap
		assert.True(t, errors.As(err, &target))
		assert.Equal(t, "test error", target.Message)
	})
}

func TestEnvMap_Validate_ErrorMessages(t *testing.T) {
	t.Run("Error message contains field names for missing vars", func(t *testing.T) {
		em := &EnvMap{
			// Only set some required fields, leave others empty
			ProjectName:  "test",
			ProjectStage: "dev",
			// Leave others empty
		}

		err := em.Validate()
		require.Error(t, err)

		var envMapErr *ErrorEnvMap
		require.True(t, errors.As(err, &envMapErr))
		assert.Contains(t, envMapErr.Message, "Vars not set:")
		assert.Contains(t, envMapErr.Message, "ARGOCD_WIZARD_ACCOUNT_PASSWORD")
	})

	t.Run("Error message contains field names for default values", func(t *testing.T) {
		em := &EnvMap{
			ProjectName:                 "<...>",
			ProjectStage:                "dev",
			ArgocdWizardAccountPassword: "test",
			ArgocdHelmRepoUsername:      "test",
			ArgocdHelmRepoPassword:      "test",
			ArgocdHelmRepoUrl:           "test",
			ArgocdGitHttpsUrl:           "test",
		}

		err := em.Validate()
		require.Error(t, err)

		var envMapErr *ErrorEnvMap
		require.True(t, errors.As(err, &envMapErr))
		assert.Contains(t, envMapErr.Message, "Vars are set to default:")
		assert.Contains(t, envMapErr.Message, "PROJECT_NAME")
	})
}

func TestEnvMap_GitAuthMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want string
	}{
		{name: "empty defaults to https", mode: "", want: GitAuthModeHTTPS},
		{name: "legacy placeholder defaults to https", mode: "<...>", want: GitAuthModeHTTPS},
		{name: "uppercase is normalized", mode: "SSH", want: GitAuthModeSSH},
		{name: "github-app is preserved", mode: "github-app", want: GitAuthModeGitHubApp},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := &EnvMap{ArgocdGitAuthMode: tt.mode}
			assert.Equal(t, tt.want, em.GitAuthMode())
		})
	}
}

func TestEnvMap_GitRepositoryURL_PrefersGitURL(t *testing.T) {
	em := &EnvMap{ArgocdGitUrl: "https://new.example.com/repo.git", ArgocdGitHttpsUrl: "https://legacy.example.com/repo.git"}
	assert.Equal(t, "https://new.example.com/repo.git", em.GitRepositoryURL())

	legacy := &EnvMap{ArgocdGitHttpsUrl: "https://legacy.example.com/repo.git"}
	assert.Equal(t, "https://legacy.example.com/repo.git", legacy.GitRepositoryURL())
}

func TestEnvMap_ValidateGitAuth(t *testing.T) {
	base := func() *EnvMap {
		return &EnvMap{
			ProjectName:                 "test-project",
			ProjectStage:                "dev",
			ArgocdWizardAccountPassword: "password123",
		}
	}

	tests := []struct {
		name    string
		mutate  func(*EnvMap)
		wantErr error
	}{
		{
			name: "https without credentials is allowed (public repo)",
			mutate: func(em *EnvMap) {
				em.ArgocdGitAuthMode = "https"
				em.ArgocdGitHttpsUrl = "https://github.com/example/repo.git"
			},
		},
		{
			name:    "https without any URL fails",
			mutate:  func(em *EnvMap) { em.ArgocdGitAuthMode = "https" },
			wantErr: ErrEnvsNotSet,
		},
		{
			name:    "https with an ssh URL fails",
			mutate:  func(em *EnvMap) { em.ArgocdGitAuthMode = "https"; em.ArgocdGitUrl = "git@github.com:example/repo.git" },
			wantErr: ErrInvalidEnvValue,
		},
		{
			name: "ssh with private key and ssh URL passes",
			mutate: func(em *EnvMap) {
				em.ArgocdGitAuthMode = "ssh"
				em.ArgocdGitUrl = "git@github.com:example/repo.git"
				em.ArgocdGitSshPrivateKey = "PRIVATE-KEY"
			},
		},
		{
			name: "ssh with non-ssh URL fails",
			mutate: func(em *EnvMap) {
				em.ArgocdGitAuthMode = "ssh"
				em.ArgocdGitUrl = "https://github.com/example/repo.git"
				em.ArgocdGitSshPrivateKey = "PRIVATE-KEY"
			},
			wantErr: ErrInvalidEnvValue,
		},
		{
			name: "github-app with all fields passes",
			mutate: func(em *EnvMap) {
				em.ArgocdGitAuthMode = "github-app"
				em.ArgocdGitUrl = "https://github.com/example/repo.git"
				em.ArgocdGitGithubAppID = "123"
				em.ArgocdGitGithubAppInstallationID = "456"
				em.ArgocdGitGithubAppPrivateKey = "PRIVATE-KEY"
			},
		},
		{
			name: "github-app missing installation id fails",
			mutate: func(em *EnvMap) {
				em.ArgocdGitAuthMode = "github-app"
				em.ArgocdGitUrl = "https://github.com/example/repo.git"
				em.ArgocdGitGithubAppID = "123"
				em.ArgocdGitGithubAppPrivateKey = "PRIVATE-KEY"
			},
			wantErr: ErrEnvsNotSet,
		},
		{
			name:    "unknown auth mode fails",
			mutate:  func(em *EnvMap) { em.ArgocdGitAuthMode = "totally-unknown" },
			wantErr: ErrInvalidEnvValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := base()
			tt.mutate(em)
			err := em.Validate()
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.wantErr), "expected %v, got %v", tt.wantErr, err)
		})
	}
}

func TestIsConfiguredEnvValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "Configured URL", value: "https://charts.example.com", want: true},
		{name: "Empty value", value: "", want: false},
		{name: "Whitespace only", value: "   ", want: false},
		{name: "Legacy placeholder", value: "<...>", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsConfiguredEnvValue(tt.value))
		})
	}
}

func TestNormalizeHelmRepoURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "Keeps HTTPS URL unchanged", value: "https://charts.example.com", want: "https://charts.example.com"},
		{name: "Strips oci scheme", value: "oci://registry-1.docker.io/bitnamicharts", want: "registry-1.docker.io/bitnamicharts"},
		{name: "Handles uppercase oci scheme", value: "OCI://registry.example.com/charts", want: "registry.example.com/charts"},
		{name: "Trims whitespace", value: "  https://charts.example.com  ", want: "https://charts.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeHelmRepoURL(tt.value))
		})
	}
}

func TestIsOCIHelmRepoURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "HTTPS repo is not OCI", value: "https://charts.example.com", want: false},
		{name: "HTTP repo is not OCI", value: "http://charts.example.com", want: false},
		{name: "Plain registry is OCI", value: "registry-1.docker.io/bitnamicharts", want: true},
		{name: "OCI scheme is OCI", value: "oci://registry-1.docker.io/bitnamicharts", want: true},
		{name: "Empty value is not OCI", value: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsOCIHelmRepoURL(tt.value))
		})
	}
}
