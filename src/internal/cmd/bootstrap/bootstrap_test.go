package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestOverlayValuesForChartIncludesValuesYaml(t *testing.T) {
	tempDir := t.TempDir()
	opts := &Options{
		OverlayValues: tempDir,
		ClusterName:   "test-cluster",
	}

	valuesPaths := overlayValuesForChart(opts, "argo-cd")

	assert.Equal(t, []string{
		filepath.Join(tempDir, "helm", "test-cluster", "argo-cd", "values.yaml"),
	}, valuesPaths)
}

func TestOverlayValuesForChartIncludesAdditionalValuesWhenPresent(t *testing.T) {
	tempDir := t.TempDir()
	chartDir := filepath.Join(tempDir, "helm", "test-cluster", "argo-cd")
	require.NoError(t, os.MkdirAll(chartDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(chartDir, "additional-values.yaml"), []byte("argo-cd: {}\n"), 0o644))

	opts := &Options{
		OverlayValues: tempDir,
		ClusterName:   "test-cluster",
	}

	valuesPaths := overlayValuesForChart(opts, "argo-cd")

	assert.Equal(t, []string{
		filepath.Join(chartDir, "values.yaml"),
		filepath.Join(chartDir, "additional-values.yaml"),
	}, valuesPaths)
}
