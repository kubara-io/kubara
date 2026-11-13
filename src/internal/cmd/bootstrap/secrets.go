package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/k8s"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/Masterminds/sprig/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	externalsecretsv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/yaml"
)

// SecretManager handles Kubernetes secret creation programmatically
type SecretManager struct {
	client *k8s.Client
}

// NewSecretManager creates a new secret manager
func NewSecretManager(client *k8s.Client) *SecretManager {
	return &SecretManager{client: client}
}

// CreateHubSecrets creates all hub cluster secrets
func (sm *SecretManager) CreateHubSecrets(ctx context.Context, o *Options) error {
	k8sOpts := k8s.DefaultApplyOptions()
	k8sOpts.FieldManager = "kubara-hub-secrets"
	k8sOpts.DryRun = o.DryRun

	var manifest []byte
	imagePullSecret, err := sm.createImagePullSecret(o.Credentials, argocdNamespace)
	if err != nil {
		return fmt.Errorf("create image pull secret: %w", err)
	}
	secrets := []*corev1.Secret{
		sm.createGitRepositorySecret(o.ClusterConfig, o.Credentials),
		imagePullSecret,
		sm.createHelmRepositorySecret(o.ClusterConfig, o.Credentials),
	}

	for _, secret := range secrets {
		if secret == nil {
			continue
		}

		yamlData, err := yaml.Marshal(secret)
		if err != nil {
			return fmt.Errorf("marshal secret: %w", err)
		}

		manifest = append(manifest, yamlData...)
		manifest = append(manifest, []byte("---\n")...)
	}

	// Handle clusterSecretStore separately since it is its own type
	clusterSecStore, err := sm.createClusterSecretStore(o)
	if err != nil {
		return fmt.Errorf("create ClusterSecretStore: %w", err)
	}
	if clusterSecStore != nil {
		yamlData, err := yaml.Marshal(clusterSecStore)
		if err != nil {
			return fmt.Errorf("marshal ClusterSecretStore: %w", err)
		}
		manifest = append(manifest, yamlData...)
	}

	if len(manifest) == 0 {
		log.Info().Msg("No secrets to create")
		return nil
	}

	// Apply all secrets in one API call
	if err := sm.client.ApplyManifest(ctx, manifest, k8sOpts); err != nil {
		return fmt.Errorf("applying hub secrets: %w", err)
	}

	log.Info().Msg("Created hub secrets successfully")
	return nil
}

// createGitRepositorySecret creates the ArgoCD git repository secret
func (sm *SecretManager) createGitRepositorySecret(cluster *config.Cluster, creds *Credentials) *corev1.Secret {
	if cluster == nil || cluster.ArgoCD == nil {
		return nil
	}

	gitURL := ""
	if cluster.ArgoCD.Repo.Git != nil {
		gitURL = cluster.ArgoCD.Repo.Git.Configs.URL
	}
	projectName := fmt.Sprintf("%s-%s", cluster.Name, cluster.Stage)

	secretName := "https-init-repo-access"
	stringData := map[string]string{
		"enableLfs": "true",
		"insecure":  "false",
		"project":   projectName,
		"type":      "git",
		"url":       gitURL,
	}

	if creds != nil {
		switch {
		case creds.GitSSHPrivateKey != "" || utils.IsSSHGitURL(gitURL):
			secretName = "ssh-init-repo-access"
			stringData["sshPrivateKey"] = creds.GitSSHPrivateKey
		case creds.GitGitHubAppID != "":
			secretName = "github-app-init-repo-access"
			stringData["githubAppID"] = creds.GitGitHubAppID
			stringData["githubAppInstallationID"] = creds.GitGitHubAppInstallationID
			stringData["githubAppPrivateKey"] = creds.GitGitHubAppPrivateKey
			if creds.GitGitHubAppEnterpriseBaseURL != "" {
				stringData["githubAppEnterpriseBaseUrl"] = creds.GitGitHubAppEnterpriseBaseURL
			}
		default:
			if creds.GitUsername != "" && creds.GitPassword != "" {
				stringData["username"] = creds.GitUsername
				stringData["password"] = creds.GitPassword
				stringData["forceHttpBasicAuth"] = "true"
			}
		}
	} else if utils.IsSSHGitURL(gitURL) {
		secretName = "ssh-init-repo-access"
	}
	stringData["name"] = secretName

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: argocdNamespace,
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type": "repository",
			},
			Annotations: map[string]string{
				"managed-by": "argocd.argoproj.io",
			},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: stringData,
	}
}

// createImagePullSecret creates the docker registry pull secret
func (sm *SecretManager) createImagePullSecret(creds *Credentials, namespace string) (*corev1.Secret, error) {
	if creds == nil || creds.DockerconfigBase64 == "" {
		return nil, nil
	}

	secretString, err := utils.DecodeB64(creds.DockerconfigBase64)
	if err != nil {
		return nil, fmt.Errorf("decode DOCKERCONFIG_BASE64: %w", err)
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "image-pull-secret",
			Namespace: namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		StringData: map[string]string{
			corev1.DockerConfigJsonKey: secretString,
		},
	}, nil
}

// createHelmRepositorySecret creates the Helm repository secret
func (sm *SecretManager) createHelmRepositorySecret(cluster *config.Cluster, creds *Credentials) *corev1.Secret {
	if cluster == nil || cluster.ArgoCD == nil || cluster.ArgoCD.HelmRepo == nil || strings.TrimSpace(cluster.ArgoCD.HelmRepo.URL) == "" {
		return nil
	}
	helmRepoURL := utils.NormalizeHelmRepoURL(cluster.ArgoCD.HelmRepo.URL)
	stringData := map[string]string{
		"url":      helmRepoURL,
		"name":     "helm-chart-repository",
		"password": "",
		"project":  fmt.Sprintf("%s-%s", cluster.Name, cluster.Stage),
		"type":     "helm",
		"username": "",
	}
	if creds != nil {
		stringData["username"] = creds.HelmRepoUsername
		stringData["password"] = creds.HelmRepoPassword
	}
	if utils.IsOCIHelmRepoURL(cluster.ArgoCD.HelmRepo.URL) {
		stringData["enableOCI"] = "true"
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "helm-charts-repository",
			Namespace: argocdNamespace,
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type": "repository",
			},
			Annotations: map[string]string{
				"managed-by": "argocd.argoproj.io",
			},
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: stringData,
	}
}

// loadAndValidateClusterSecretStore reads a ClusterSecretStore manifest file,
// processes it through template engine with cluster config, and validates the structure
func loadAndValidateClusterSecretStore(filePath string, cluster *config.Cluster) (*externalsecretsv1.ClusterSecretStore, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read ClusterSecretStore file: %w", err)
	}

	// Build template context following the same pattern as buildTemplateContext in generate.go
	// Convert struct to JSON to get lowercase/camelCase keys from json tags
	clusterJSON, err := json.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("marshal cluster to JSON: %w", err)
	}

	// Unmarshal back to map with lowercase keys
	var clusterMap map[string]interface{}
	if err := json.Unmarshal(clusterJSON, &clusterMap); err != nil {
		return nil, fmt.Errorf("unmarshal cluster JSON to map: %w", err)
	}

	// Wrap in map with "cluster" key for template access
	templateData := map[string]interface{}{
		"cluster": clusterMap,
	}

	// Process through template engine (works for both .yaml and .tmpl files)
	// Cluster fields are accessible as {{ .cluster.name }}, {{ .cluster.stage }}, {{ .cluster.dnsName }}, etc.
	tmpl, err := template.New("clustersecretstore").Funcs(sprig.FuncMap()).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	// Unmarshal into ClusterSecretStore struct
	var css externalsecretsv1.ClusterSecretStore
	if err := yaml.Unmarshal(buf.Bytes(), &css); err != nil {
		return nil, fmt.Errorf("unmarshal ClusterSecretStore: %w", err)
	}

	// Validate Kind
	if css.Kind != "ClusterSecretStore" {
		return nil, fmt.Errorf("invalid kind: expected ClusterSecretStore, got %q", css.Kind)
	}

	return &css, nil
}

// createClusterSecretStore creates the ClusterSecretStore for external-secrets
func (sm *SecretManager) createClusterSecretStore(o *Options) (*externalsecretsv1.ClusterSecretStore, error) {
	if o.WithESCSSPath == "" {
		log.Warn().Msg("ClusterSecretStore manifest file not provided. You must apply an existing ClusterSecretStore or create one manually")
		return nil, nil
	}

	if o.ClusterConfig == nil {
		return nil, fmt.Errorf("cluster config is required when using ClusterSecretStore manifest file")
	}

	css, err := loadAndValidateClusterSecretStore(o.WithESCSSPath, o.ClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("load ClusterSecretStore: %w", err)
	}

	// Validate that the name follows the expected pattern: <cluster-name>-<cluster-stage>
	expectedName := fmt.Sprintf("%s-%s", o.ClusterConfig.Name, o.ClusterConfig.Stage)
	if css.Name != expectedName {
		log.Warn().
			Str("expected", expectedName).
			Str("actual", css.Name).
			Msg("ClusterSecretStore name does not follow the pattern <cluster.name>-<cluster.stage>. Make sure to update any helm values reffering to the ClusterSecretStore accordingly")
	}

	log.Info().
		Str("file", o.WithESCSSPath).
		Str("cluster", o.ClusterConfig.Name).
		Str("name", css.Name).
		Msg("Loaded ClusterSecretStore from file")
	return css, nil
}
