package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/core/v1"
)

const completionMessageTemplate = `
🎉 ArgoCD bootstrap complete!

You can access the Argo CD UI with user "wizard" and your chosen password "%s" at:

    kubectl port-forward svc/argocd-server -n argocd 8080:443 --kubeconfig ...

Then open: http://localhost:8080/argocd%s

📝 Next steps:
1. Log in with username: wizard
2. Configure your applications
3. Set up monitoring and logging as needed`

func Test_UsesClusterDNSNameForIngressURL(t *testing.T) {
	config := CompletionLogConfig{}
	config.WizardPassword = "wizard_password"
	config.ClusterDNSName = "cluster.example.com"

	expected := fmt.Sprintf(completionMessageTemplate, config.WizardPassword,
		" or try: https://cluster.example.com/argocd (if ingress is running)")
	actual := CreateCompletionMessage(config)
	assert.Equal(t, expected, actual)
}

func Test_MissingEnvVariableLeadsToURLBeingOmitted(t *testing.T) {
	config := CompletionLogConfig{}

	config.WizardPassword = "wizard_password"

	expected := fmt.Sprintf(completionMessageTemplate, config.WizardPassword, "")
	actual := CreateCompletionMessage(config)

	assert.Equal(t, expected, actual)
}

func TestLocalCompletionMessageUsesWizardLoginOnly(t *testing.T) {
	config := CompletionLogConfig{
		Local:          true,
		ClusterDNSName: "127.0.0.1.traefik.me",
		WizardPassword: "magic",
		OpenBaoHost:    "openbao.127.0.0.1.traefik.me",
	}

	actual := CreateCompletionMessage(config)

	assert.Contains(t, actual, "wizard / magic")
	assert.NotContains(t, actual, "OpenBao-backed SSO via Dex")
	assert.Contains(t, actual, "https://openbao.127.0.0.1.traefik.me/ui")
	assert.NotContains(t, actual, "login with root")
	assert.Contains(t, actual, "Retrieve the generated local OpenBao root token")
}

func TestBuildLocalTraefikBootstrapServiceMatchesHelmOwnershipMetadata(t *testing.T) {
	service := buildLocalTraefikBootstrapService()

	assert.Equal(t, localTraefikReleaseName, service.Name)
	assert.Equal(t, localTraefikNamespace, service.Namespace)
	assert.Equal(t, v1.ServiceTypeLoadBalancer, service.Spec.Type)
	assert.Equal(t, "Helm", service.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "traefik-traefik", service.Labels["app.kubernetes.io/instance"])
	assert.Equal(t, localTraefikReleaseName, service.Annotations["meta.helm.sh/release-name"])
	assert.Equal(t, localTraefikNamespace, service.Annotations["meta.helm.sh/release-namespace"])
}

func TestOverlayValuesForChartIncludesGeneratedValuesYaml(t *testing.T) {
	tempDir := t.TempDir()
	opts := &Options{
		PlatformConfigs: tempDir,
		ClusterName:     "test-cluster",
	}

	valuesPaths := overlayValuesForChart(opts, "argo-cd")

	assert.Equal(t, []string{
		filepath.Join(tempDir, "test-cluster", "helm", "argo-cd", "values.generated.yaml"),
	}, valuesPaths)
}

func TestOverlayValuesForChartIncludesExtraValuesFilesInLexicalOrder(t *testing.T) {
	tempDir := t.TempDir()
	chartDir := filepath.Join(tempDir, "test-cluster", "helm", "argo-cd")
	require.NoError(t, os.MkdirAll(chartDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(chartDir, "values-z.yaml"), []byte("z: true\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(chartDir, "values-additional.yaml"), []byte("argo-cd: {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(chartDir, "values-a.yaml"), []byte("a: true\n"), 0o644))

	opts := &Options{
		PlatformConfigs: tempDir,
		ClusterName:     "test-cluster",
	}

	valuesPaths := overlayValuesForChart(opts, "argo-cd")

	assert.Equal(t, []string{
		filepath.Join(chartDir, "values.generated.yaml"),
		filepath.Join(chartDir, "values-a.yaml"),
		filepath.Join(chartDir, "values-additional.yaml"),
		filepath.Join(chartDir, "values-z.yaml"),
	}, valuesPaths)
}

func TestLocalOpenBaoExecArgsPreservesSecretValues(t *testing.T) {
	args := localOpenBaoExecArgs("root-token", false,
		"bao", "kv", "put", "kv/example",
		"password=review$UNSET",
		"command=$(id)",
		"multiline=first\nsecond",
	)

	assert.Equal(t, []string{
		"-n", localOpenBaoNamespace,
		"exec",
		localOpenBaoPodName,
		"-c", "openbao",
		"--",
		"env", "BAO_TOKEN=root-token",
		"bao", "kv", "put", "kv/example",
		"password=review$UNSET",
		"command=$(id)",
		"multiline=first\nsecond",
	}, args)
	assert.NotContains(t, args, "sh")
}

func TestRedactOpenBaoToken(t *testing.T) {
	commandErr := errors.New("env BAO_TOKEN=root-token bao kv put failed: root-token")
	err := fmt.Errorf("execute local OpenBao command: %w", &redactedOpenBaoCommandError{
		err:   commandErr,
		token: "root-token",
	})

	assert.Equal(t, "execute local OpenBao command: env BAO_TOKEN=<redacted> bao kv put failed: <redacted>", err.Error())
	assert.ErrorIs(t, err, commandErr)
}

func TestWriteLocalOpenBaoValuesUsesChartServerImageForAutoUnsealer(t *testing.T) {
	valuesPath := filepath.Join(t.TempDir(), "values.yaml")
	state := &LocalState{
		OpenBaoValuesPath: valuesPath,
		OpenBaoHost:       "openbao.127.0.0.1.traefik.me",
	}

	require.NoError(t, writeLocalOpenBaoValues(state))
	content, err := os.ReadFile(valuesPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), `image: '{{ .Values.server.image.registry | default "docker.io" }}/{{ .Values.server.image.repository }}:{{ .Values.server.image.tag | default (trimPrefix "v" .Chart.AppVersion) }}'`)
	assert.NotContains(t, string(content), "openbao/openbao:2.0.1")
}
