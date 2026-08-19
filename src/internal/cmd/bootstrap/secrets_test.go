package bootstrap

import (
	"context"
	"fmt"
	"testing"

	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/k8s"
	"github.com/kubara-io/kubara/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
)

func TestCreateHelmRepositorySecret(t *testing.T) {
	sm := &SecretManager{}

	t.Run("returns nil when helm repo URL is missing", func(t *testing.T) {
		secret := sm.createHelmRepositorySecret(&envconfig.EnvMap{
			ProjectName:       "test",
			ProjectStage:      "dev",
			ArgocdHelmRepoUrl: "",
		})
		assert.Nil(t, secret)
	})

	t.Run("returns nil when helm repo URL is legacy placeholder", func(t *testing.T) {
		secret := sm.createHelmRepositorySecret(&envconfig.EnvMap{
			ProjectName:       "test",
			ProjectStage:      "dev",
			ArgocdHelmRepoUrl: "<...>",
		})
		assert.Nil(t, secret)
	})

	t.Run("creates secret for classic https helm repo", func(t *testing.T) {
		secret := sm.createHelmRepositorySecret(&envconfig.EnvMap{
			ProjectName:            "test",
			ProjectStage:           "dev",
			ArgocdHelmRepoUrl:      "https://charts.example.com",
			ArgocdHelmRepoUsername: "user",
			ArgocdHelmRepoPassword: "pass",
		})

		require.NotNil(t, secret)
		assert.Equal(t, "https://charts.example.com", secret.StringData["url"])
		assert.Equal(t, "user", secret.StringData["username"])
		assert.Equal(t, "pass", secret.StringData["password"])
		_, hasEnableOCI := secret.StringData["enableOCI"]
		assert.False(t, hasEnableOCI)
	})

	t.Run("creates secret for OCI helm registry and strips oci scheme", func(t *testing.T) {
		secret := sm.createHelmRepositorySecret(&envconfig.EnvMap{
			ProjectName:       "test",
			ProjectStage:      "dev",
			ArgocdHelmRepoUrl: "oci://registry-1.docker.io/bitnamicharts",
		})

		require.NotNil(t, secret)
		assert.Equal(t, "registry-1.docker.io/bitnamicharts", secret.StringData["url"])
		assert.Equal(t, "true", secret.StringData["enableOCI"])
	})
}

type fakeInfrastructureOutputReader map[string]string

func (f fakeInfrastructureOutputReader) Read(_ context.Context, _, _, name string) (string, error) {
	value, exists := f[name]
	if !exists {
		return "", fmt.Errorf("missing output %q", name)
	}
	return value, nil
}

func TestResolveClusterSecretStore(t *testing.T) {
	existingStore := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "external-secrets.io/v1",
			"kind":       "ClusterSecretStore",
			"metadata": map[string]interface{}{
				"name": "test-cluster-local",
			},
		},
	}

	tests := []struct {
		name         string
		provider     config.TerraformProvider
		objects      []runtime.Object
		dryRun       bool
		disabled     bool
		wantCreated  bool
		wantValidate bool
		wantError    string
	}{
		{
			name:     "accepts existing store",
			provider: config.TerraformProviderStackit,
			objects:  []runtime.Object{existingStore},
		},
		{
			name:         "creates missing STACKIT store",
			provider:     config.TerraformProviderStackit,
			wantCreated:  true,
			wantValidate: true,
		},
		{
			name:     "allows generated T Cloud Public store",
			provider: config.TerraformProviderTCloudPublic,
		},
		{
			name:     "skips live validation during dry-run",
			provider: config.TerraformProviderStackit,
			dryRun:   true,
		},
		{
			name:     "skips validation when external secrets is disabled",
			provider: config.TerraformProviderStackit,
			disabled: true,
		},
		{
			name:      "rejects missing store for unsupported provider",
			provider:  config.TerraformProviderNone,
			wantError: `ClusterSecretStore "test-cluster-local" was not found`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dynamicClient := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), tt.objects...)
			sm := NewSecretManager(&k8s.Client{
				Clientset:     kubernetesfake.NewSimpleClientset(),
				DynamicClient: dynamicClient,
			})
			sm.outputReader = fakeInfrastructureOutputReader{
				stackitVaultUsernameOutput: "readonly-user",
				stackitVaultPasswordOutput: "cmVhZG9ubHktcGFzc3dvcmQ=",
				stackitVaultAPIURLOutput:   "https://secrets.example.com",
				stackitVaultPathOutput:     "instance-id",
			}
			sm.iacCommandResolver = func(string) (string, error) { return "tofu", nil }
			externalSecretsStatus := service.StatusEnabled
			if tt.disabled {
				externalSecretsStatus = service.StatusDisabled
			}
			opts := &Options{
				ClusterConfig: &config.Cluster{
					Name:  "test-cluster",
					Stage: "local",
					Terraform: &config.Terraform{
						Provider: tt.provider,
					},
					Services: service.Services{
						"external-secrets": {Status: externalSecretsStatus},
					},
				},
				DryRun: tt.dryRun,
			}

			css, credentialSecret, validate, err := sm.resolveClusterSecretStore(context.Background(), opts)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantValidate, validate)
			if !tt.wantCreated {
				assert.Nil(t, css)
				assert.Nil(t, credentialSecret)
				return
			}

			require.NotNil(t, css)
			require.NotNil(t, credentialSecret)
			assert.Equal(t, "test-cluster-local", css.Name)
			assert.Equal(t, "readonly-user", credentialSecret.StringData["username"])
			assert.Equal(t, "readonly-password", credentialSecret.StringData["password"])
			require.NotNil(t, css.Spec.Provider)
			require.NotNil(t, css.Spec.Provider.Vault)
			assert.Equal(t, "https://secrets.example.com", css.Spec.Provider.Vault.Server)
			assert.Equal(t, "instance-id", *css.Spec.Provider.Vault.Path)
			assert.Equal(t, "readonly-user", css.Spec.Provider.Vault.Auth.UserPass.Username)
		})
	}
}

func TestResolveClusterSecretStoreRequiresClusterConfig(t *testing.T) {
	sm := NewSecretManager(&k8s.Client{
		Clientset:     kubernetesfake.NewSimpleClientset(),
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
	})

	_, _, _, err := sm.resolveClusterSecretStore(context.Background(), &Options{})

	require.EqualError(t, err, "cluster config is required")
}

func TestCreateStackitClusterSecretStoreDoesNotExposeInvalidPasswordOutput(t *testing.T) {
	sm := NewSecretManager(&k8s.Client{
		Clientset:     kubernetesfake.NewSimpleClientset(),
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
	})
	sm.outputReader = fakeInfrastructureOutputReader{
		stackitVaultUsernameOutput: "readonly-user",
		stackitVaultPasswordOutput: "sensitive-invalid-value",
	}
	sm.iacCommandResolver = func(string) (string, error) { return "terraform", nil }
	opts := &Options{
		PlatformConfigs: t.TempDir(),
		ClusterConfig: &config.Cluster{
			Name:  "test-cluster",
			Stage: "local",
		},
	}

	_, _, err := sm.createStackitClusterSecretStore(context.Background(), opts)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "sensitive-invalid-value")
}

func TestValidateCreatedExternalSecretsResources(t *testing.T) {
	existingStore := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "external-secrets.io/v1",
			"kind":       "ClusterSecretStore",
			"metadata": map[string]interface{}{
				"name": "test-cluster-local",
			},
		},
	}
	credentialSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stackitCredentialSecretName,
			Namespace: externalSecretsNamespace,
		},
	}
	sm := NewSecretManager(&k8s.Client{
		Clientset:     kubernetesfake.NewSimpleClientset(credentialSecret),
		DynamicClient: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), existingStore),
	})

	require.NoError(t, sm.validateCredentialSecret(context.Background(), stackitCredentialSecretName))
	require.NoError(t, sm.validateClusterSecretStore(context.Background(), "test-cluster-local"))
	require.ErrorContains(t, sm.validateCredentialSecret(context.Background(), "missing"), "validate credential Secret")
	require.ErrorContains(t, sm.validateClusterSecretStore(context.Background(), "missing"), "was not found after apply")
}
