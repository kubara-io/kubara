package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/template"

	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
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

// CreateControlPlaneSecrets creates all control plane secrets
func (sm *SecretManager) CreateControlPlaneSecrets(ctx context.Context, o *Options) error {
	k8sOpts := k8s.DefaultApplyOptions()
	k8sOpts.FieldManager = "kubara-controlplane-secrets"
	k8sOpts.DryRun = o.DryRun

	var manifest []byte
	secrets := []*corev1.Secret{
		sm.createGitRepositorySecret(o.EnvMap),
		sm.createImagePullSecret(o.EnvMap, argocdNamespace),
		sm.createHelmRepositorySecret(o.EnvMap),
	}

	for _, secret := range secrets {
		if secret == nil {
			continue
		}

		yamlData, err := yaml.Marshal(secret)
		if err != nil {
			return fmt.Errorf("marshaling secret: %w", err)
		}

		manifest = append(manifest, yamlData...)
		manifest = append(manifest, []byte("---\n")...)
	}

	// Handle clusterSecretStore separately since it is its own type
	clusterSecStore, err := sm.createClusterSecretStore(o)
	if err != nil {
		return fmt.Errorf("creating ClusterSecretStore: %w", err)
	}
	if clusterSecStore != nil {
		yamlData, err := yaml.Marshal(clusterSecStore)
		if err != nil {
			return fmt.Errorf("marshaling ClusterSecretStore: %w", err)
		}
		manifest = append(manifest, yamlData...)
	}

	if len(manifest) == 0 {
		log.Info().Msg("No secrets to create")
		return nil
	}

	// Apply all secrets in one API call
	if err := sm.client.ApplyManifest(ctx, manifest, k8sOpts); err != nil {
		return fmt.Errorf("applying control plane secrets: %w", err)
	}

	log.Info().Msg("Created control plane secrets successfully")
	return nil
}

// createGitRepositorySecret creates the ArgoCD git repository secret
func (sm *SecretManager) createGitRepositorySecret(em *envconfig.EnvMap) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "https-init-repo-access",
			Namespace: argocdNamespace,
			Labels: map[string]string{
				"argocd.argoproj.io/secret-type": "repository",
			},
			Annotations: map[string]string{
				"managed-by": "argocd.argoproj.io",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"enableLfs":          "true",
			"forceHttpBasicAuth": "true",
			"insecure":           "false",
			"password":           em.ArgocdGitPatOrPassword,
			"username":           em.ArgocdGitUsername,
			"name":               "https-init-repo-access",
			"url":                em.ArgocdGitHttpsUrl,
			"project":            fmt.Sprintf("%s-%s", em.ProjectName, em.ProjectStage),
		},
	}
}

// createImagePullSecret creates the docker registry pull secret
func (sm *SecretManager) createImagePullSecret(em *envconfig.EnvMap, namespace string) *corev1.Secret {
	secretString, _ := utils.DecodeB64(em.DockerconfigBase64)
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
	}
}

// createHelmRepositorySecret creates the Helm repository secret
func (sm *SecretManager) createHelmRepositorySecret(em *envconfig.EnvMap) *corev1.Secret {
	if !envconfig.IsConfiguredEnvValue(em.ArgocdHelmRepoUrl) {
		return nil
	}
	helmRepoURL := envconfig.NormalizeHelmRepoURL(em.ArgocdHelmRepoUrl)
	stringData := map[string]string{
		"url":      helmRepoURL,
		"name":     "helm-chart-repository",
		"password": em.ArgocdHelmRepoPassword,
		"project":  fmt.Sprintf("%s-%s", em.ProjectName, em.ProjectStage),
		"type":     "helm",
		"username": em.ArgocdHelmRepoUsername,
	}
	if envconfig.IsOCIHelmRepoURL(em.ArgocdHelmRepoUrl) {
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
		return nil, fmt.Errorf("reading ClusterSecretStore file: %w", err)
	}

	// Build template context following the same pattern as buildTemplateContext in generate.go
	// Convert struct to JSON to get lowercase/camelCase keys from json tags
	clusterJSON, err := json.Marshal(cluster)
	if err != nil {
		return nil, fmt.Errorf("marshaling cluster to JSON: %w", err)
	}

	// Unmarshal back to map with lowercase keys
	var clusterMap map[string]interface{}
	if err := json.Unmarshal(clusterJSON, &clusterMap); err != nil {
		return nil, fmt.Errorf("unmarshaling cluster JSON to map: %w", err)
	}

	// Wrap in map with "cluster" key for template access
	templateData := map[string]interface{}{
		"cluster": clusterMap,
	}

	// Process through template engine (works for both .yaml and .tmpl files)
	// Cluster fields are accessible as {{ .cluster.name }}, {{ .cluster.stage }}, {{ .cluster.dnsName }}, etc.
	tmpl, err := template.New("clustersecretstore").Funcs(sprig.FuncMap()).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	// Unmarshal into ClusterSecretStore struct
	var css externalsecretsv1.ClusterSecretStore
	if err := yaml.Unmarshal(buf.Bytes(), &css); err != nil {
		return nil, fmt.Errorf("unmarshaling ClusterSecretStore: %w", err)
	}

	// Validate Kind
	if css.Kind != "ClusterSecretStore" {
		return nil, fmt.Errorf("invalid Kind: expected ClusterSecretStore, got %s", css.Kind)
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
		return nil, fmt.Errorf("loading ClusterSecretStore from %s: %w", o.WithESCSSPath, err)
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
