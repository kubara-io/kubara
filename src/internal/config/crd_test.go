package config

import (
	"bytes"
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
