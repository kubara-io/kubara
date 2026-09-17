package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_RejectsServiceDefinitionWithUnknownFields(t *testing.T) {
	tempDir := t.TempDir()
	catDir := filepath.Join(tempDir, "broken-catalog")
	require.NoError(t, os.MkdirAll(filepath.Join(catDir, "services"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(catDir, "Catalog.yaml"), []byte(`apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: broken
spec:
  version: 1.0.0
`), 0o600))

	require.NoError(t, os.WriteFile(filepath.Join(catDir, "services", "bad.yaml"), []byte(`apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: bad-service
spec:
  chartpath: bad-service
  status: enabled
`), 0o600))

	_, err := Load(LoadOptions{
		BootstrapCatalog: catDir,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chartpath")
}
