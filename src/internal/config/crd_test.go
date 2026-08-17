package config

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubara-io/kubara/internal/catalog"

	"github.com/kubara-io/libkubara/crdvalidate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatformSetupCRDCompilesWithSelectedKubernetesVersion(t *testing.T) {
	validator, err := crdvalidate.Compile(bytes.NewReader(PlatformSetupCRD()))
	require.NoError(t, err)
	require.NotNil(t, validator)

	definition, err := crdvalidate.DecodeCRDBytes(PlatformSetupCRD())
	require.NoError(t, err)
	validatorFromDef, err := definition.Compile()
	require.NoError(t, err)
	require.NotNil(t, validatorFromDef)
}

func TestConfigurationJSONSchemaIdentifiesTheCustomResource(t *testing.T) {
	schema, err := ConfigurationJSONSchema()
	require.NoError(t, err)
	assert.Equal(t, PlatformSetupKind, schema["title"])
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, properties, "apiVersion")
	assert.Contains(t, properties, "kind")
	assert.Contains(t, properties, "metadata")
}

func TestConfigStoreLoadsNormalizedPlatformSetup(t *testing.T) {
	legacy := createLoadedConfigStore(t, newValidTestConfig())
	data, err := platformSetupDocument(legacy.GetConfig(), "platform")
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "platform-setup.yaml")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	store := NewConfigStore(filepath.Dir(path), path, catalog.LoadOptions{})
	require.NoError(t, store.Load())
	assert.Equal(t, legacy.GetConfig(), store.GetConfig())
}

func TestPlatformSetupRejectsUnknownFields(t *testing.T) {
	document := []byte(`apiVersion: kubara.io/v1alpha5
kind: PlatformSetup
metadata:
  name: platform
spec:
  clusters: []
  unsupported: true
`)
	_, err := decodePlatformSetup(document)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestValidatePlatformSetupTransition(t *testing.T) {
	old := []byte(`apiVersion: kubara.io/v1alpha5
kind: PlatformSetup
metadata:
  name: platform
spec:
  clusters: []
`)
	proposed := []byte(`apiVersion: kubara.io/v1alpha5
kind: PlatformSetup
metadata:
  name: platform
spec:
  bootstrapCatalog: oci://example.invalid/catalog:v1
  clusters: []
`)
	cfg, err := ValidatePlatformSetupTransition(context.Background(), old, proposed)
	require.NoError(t, err)
	assert.Equal(t, "oci://example.invalid/catalog:v1", *cfg.BootstrapCatalog)

	_, err = ValidatePlatformSetupTransition(context.Background(), old, bytes.Replace(proposed, []byte("name: platform"), []byte("name: other"), 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transition")
}
