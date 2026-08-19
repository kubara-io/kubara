package bootstrap

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"

	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/service"

	externalsecretsv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	externalsecretsmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	externalSecretsNamespace    = "external-secrets"
	stackitCredentialSecretName = "stackit-secrets-manager-cred"
	stackitVaultUsernameOutput  = "vault_user_ro_name"
	stackitVaultPasswordOutput  = "vault_user_ro_password_b64"
	stackitVaultAPIURLOutput    = "vault_api_url"
	stackitVaultPathOutput      = "vault_path"
)

var clusterSecretStoreGVR = schema.GroupVersionResource{
	Group:    "external-secrets.io",
	Version:  "v1",
	Resource: "clustersecretstores",
}

func (sm *SecretManager) resolveClusterSecretStore(ctx context.Context, o *Options) (*externalsecretsv1.ClusterSecretStore, *corev1.Secret, bool, error) {
	if o.ClusterConfig == nil {
		return nil, nil, false, fmt.Errorf("cluster config is required")
	}

	externalSecrets, exists := o.ClusterConfig.Services["external-secrets"]
	if !exists || externalSecrets.Status != service.StatusEnabled {
		return nil, nil, false, nil
	}

	if o.ClusterConfig.Terraform != nil && o.ClusterConfig.Terraform.Provider == config.TerraformProviderTCloudPublic {
		log.Info().Msg("ClusterSecretStore is managed by the generated T Cloud Public configuration")
		return nil, nil, false, nil
	}

	if o.DryRun {
		log.Warn().
			Str("name", clusterSecretStoreName(o.ClusterConfig)).
			Msg("Skipping ClusterSecretStore discovery and automatic creation during dry-run")
		return nil, nil, false, nil
	}

	exists, err := sm.clusterSecretStoreExists(ctx, clusterSecretStoreName(o.ClusterConfig))
	if err != nil {
		return nil, nil, false, err
	}
	if exists {
		log.Info().Str("name", clusterSecretStoreName(o.ClusterConfig)).Msg("Using existing ClusterSecretStore")
		return nil, nil, false, nil
	}

	if o.ClusterConfig.Terraform != nil && o.ClusterConfig.Terraform.Provider == config.TerraformProviderStackit {
		css, credentialSecret, err := sm.createStackitClusterSecretStore(ctx, o)
		return css, credentialSecret, err == nil, err
	}

	return nil, nil, false, fmt.Errorf("ClusterSecretStore %q was not found; provide it with --with-es-css-file or create it in the cluster and rerun bootstrap", clusterSecretStoreName(o.ClusterConfig))
}

func (sm *SecretManager) createStackitClusterSecretStore(ctx context.Context, o *Options) (*externalsecretsv1.ClusterSecretStore, *corev1.Secret, error) {
	if sm.iacCommandResolver == nil {
		return nil, nil, fmt.Errorf("infrastructure command resolver is required")
	}
	command, err := sm.iacCommandResolver(o.IaCCommand)
	if err != nil {
		return nil, nil, err
	}
	if sm.outputReader == nil {
		return nil, nil, fmt.Errorf("infrastructure output reader is required")
	}
	if sm.client == nil || sm.client.Clientset == nil {
		return nil, nil, fmt.Errorf("kubernetes clientset is required")
	}

	infrastructureDir := filepath.Join(o.PlatformConfigs, o.ClusterConfig.Name, "terraform", "infrastructure")
	username, err := sm.outputReader.Read(ctx, infrastructureDir, command, stackitVaultUsernameOutput)
	if err != nil {
		return nil, nil, err
	}
	passwordBase64, err := sm.outputReader.Read(ctx, infrastructureDir, command, stackitVaultPasswordOutput)
	if err != nil {
		return nil, nil, err
	}
	passwordBytes, err := base64.StdEncoding.DecodeString(passwordBase64)
	if err != nil {
		return nil, nil, fmt.Errorf("decode %s: %w", stackitVaultPasswordOutput, err)
	}
	password := string(passwordBytes)
	server, err := sm.outputReader.Read(ctx, infrastructureDir, command, stackitVaultAPIURLOutput)
	if err != nil {
		return nil, nil, err
	}
	vaultPath, err := sm.outputReader.Read(ctx, infrastructureDir, command, stackitVaultPathOutput)
	if err != nil {
		return nil, nil, err
	}

	if err := sm.client.EnsureNamespace(ctx, externalSecretsNamespace, false); err != nil {
		return nil, nil, fmt.Errorf("ensure external-secrets namespace: %w", err)
	}

	credentialSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      stackitCredentialSecretName,
			Namespace: externalSecretsNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"username": username,
			"password": password,
		},
	}

	secretNamespace := externalSecretsNamespace
	css := &externalsecretsv1.ClusterSecretStore{
		TypeMeta: metav1.TypeMeta{APIVersion: "external-secrets.io/v1", Kind: "ClusterSecretStore"},
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterSecretStoreName(o.ClusterConfig),
			Labels: map[string]string{
				"argocd.argoproj.io/instance": fmt.Sprintf("%s-external-secrets", o.ClusterConfig.Name),
			},
		},
		Spec: externalsecretsv1.SecretStoreSpec{
			Provider: &externalsecretsv1.SecretStoreProvider{
				Vault: &externalsecretsv1.VaultProvider{
					Server:  server,
					Path:    &vaultPath,
					Version: externalsecretsv1.VaultKVStoreV2,
					Auth: &externalsecretsv1.VaultAuth{
						UserPass: &externalsecretsv1.VaultUserPassAuth{
							Path:     "userpass",
							Username: username,
							SecretRef: externalsecretsmeta.SecretKeySelector{
								Name:      stackitCredentialSecretName,
								Namespace: &secretNamespace,
								Key:       "password",
							},
						},
					},
				},
			},
		},
	}

	log.Info().Msg("Prepared STACKIT ClusterSecretStore from infrastructure outputs")
	return css, credentialSecret, nil
}

func (sm *SecretManager) clusterSecretStoreExists(ctx context.Context, name string) (bool, error) {
	if sm.client == nil || sm.client.DynamicClient == nil {
		return false, fmt.Errorf("kubernetes dynamic client is required")
	}
	_, err := sm.client.DynamicClient.Resource(clusterSecretStoreGVR).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("get ClusterSecretStore %q: %w", name, err)
	}
	return true, nil
}

func (sm *SecretManager) validateCredentialSecret(ctx context.Context, name string) error {
	if _, err := sm.client.Clientset.CoreV1().Secrets(externalSecretsNamespace).Get(ctx, name, metav1.GetOptions{}); err != nil {
		return fmt.Errorf("validate credential Secret %q: %w", name, err)
	}
	return nil
}

func (sm *SecretManager) validateClusterSecretStore(ctx context.Context, name string) error {
	exists, err := sm.clusterSecretStoreExists(ctx, name)
	if err != nil {
		return fmt.Errorf("validate ClusterSecretStore: %w", err)
	}
	if !exists {
		return fmt.Errorf("validate ClusterSecretStore: %q was not found after apply", name)
	}
	return nil
}

func clusterSecretStoreName(cluster *config.Cluster) string {
	return fmt.Sprintf("%s-%s", cluster.Name, cluster.Stage)
}
