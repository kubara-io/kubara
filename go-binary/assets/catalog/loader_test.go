package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBuiltIn(t *testing.T) {
	cat, err := LoadBuiltIn()
	require.NoError(t, err)

	assert.NotEmpty(t, cat.Services)
	assert.Contains(t, cat.Services, "cert-manager")
	assert.Contains(t, cat.Services, "argo-cd")
}

func TestLoad_ExternalCollisionWithoutOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	servicesDir := filepath.Join(tempDir, "services")
	require.NoError(t, os.MkdirAll(servicesDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(servicesDir, "argo-cd.yaml"), []byte(`
apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: argo-cd
spec:
  chartPath: argo-cd
  default: enabled
`), 0644))

	_, err := Load(LoadOptions{DistributionPath: tempDir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestLoad_ExternalCollisionWithOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	servicesDir := filepath.Join(tempDir, "services")
	require.NoError(t, os.MkdirAll(servicesDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(servicesDir, "argo-cd.yaml"), []byte(`
apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: argo-cd
spec:
  chartPath: custom-argo-cd
  default: enabled
`), 0644))

	cat, err := Load(LoadOptions{DistributionPath: tempDir, Overwrite: true})
	require.NoError(t, err)
	assert.Equal(t, "custom-argo-cd", cat.Services["argo-cd"].Spec.ChartPath)
}

func TestCanonicalServiceName(t *testing.T) {
	assert.Equal(t, "cert-manager", CanonicalServiceName("certManager"))
	assert.Equal(t, "metallb", CanonicalServiceName("metalLb"))
	assert.Equal(t, "external-dns", CanonicalServiceName("external-dns"))
}
