package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/render"
	"github.com/kubara-io/kubara/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEnabledServiceTemplatePathPredicate_SkipsDisabledConfigsService(t *testing.T) {
	predicate := buildServiceTemplateFilter(
		config.Cluster{
			Services: service.Services{
				"loki": {Status: service.StatusDisabled},
			},
		},
		catalog.Catalog{
			Services: map[string]catalog.ServiceDefinition{
				"loki": {
					Spec: catalog.ServiceSpec{
						ChartPath: "loki",
					},
				},
			},
		},
	)

	assert.False(t, predicate(filepath.Join(render.DefaultPlatformConfigsPath, render.Helm.String(), "loki", "values.generated.yaml.tplt")))
}

func TestBuildEnabledServiceTemplatePathPredicate_AlwaysIncludesBootstrapServiceTemplates(t *testing.T) {
	predicate := buildServiceTemplateFilter(
		config.Cluster{},
		catalog.Catalog{
			Services: map[string]catalog.ServiceDefinition{
				"argo-cd": {
					Spec: catalog.ServiceSpec{
						ChartPath: "argo-cd",
					},
				},
			},
		},
	)

	assert.True(t, predicate(filepath.Join(render.DefaultPlatformConfigsPath, render.Helm.String(), "argo-cd", "values.generated.yaml.tplt")))
}

func TestCleanupOldFiles_OnlyDeletesOnReset(t *testing.T) {
	tempDir := t.TempDir()
	platformComponents := filepath.Join(tempDir, "platform-components")
	tfDir := filepath.Join(platformComponents, "terraform")
	helmDir := filepath.Join(platformComponents, "helm")

	require.NoError(t, os.MkdirAll(tfDir, 0o750))
	require.NoError(t, os.MkdirAll(helmDir, 0o750))

	// Without Reset: directories must NOT be deleted
	optsNoReset := &Options{
		PlatformComponents: platformComponents,
		TemplateType:       render.All,
		Reset:              false,
	}
	require.NoError(t, optsNoReset.cleanupOldFiles())
	assert.DirExists(t, tfDir)
	assert.DirExists(t, helmDir)

	// With DryRun and Reset: directories must NOT be deleted
	optsDryRun := &Options{
		PlatformComponents: platformComponents,
		TemplateType:       render.All,
		Reset:              true,
		DryRun:             true,
	}
	require.NoError(t, optsDryRun.cleanupOldFiles())
	assert.DirExists(t, tfDir)
	assert.DirExists(t, helmDir)

	// With Reset: directories MUST be deleted
	optsReset := &Options{
		PlatformComponents: platformComponents,
		TemplateType:       render.All,
		Reset:              true,
	}
	require.NoError(t, optsReset.cleanupOldFiles())
	assert.NoDirExists(t, tfDir)
	assert.NoDirExists(t, helmDir)
}

func TestWriteTemplateResults_DoesNotOverwriteExistingFiles(t *testing.T) {
	tempDir := t.TempDir()
	existingFilePath := filepath.Join(tempDir, "existing.txt")
	newFilePath := filepath.Join(tempDir, "new.txt")

	require.NoError(t, os.WriteFile(existingFilePath, []byte("original content"), 0o644))

	opts := &Options{
		DryRun: false,
	}

	results := []render.TemplateResult{
		{
			Path:    existingFilePath,
			Content: "modified content",
		},
		{
			Path:    newFilePath,
			Content: "brand new content",
		},
	}

	require.NoError(t, opts.writeTemplateResults(results))

	// Existing file must not be overwritten
	existingContent, err := os.ReadFile(existingFilePath)
	require.NoError(t, err)
	assert.Equal(t, "original content", string(existingContent))

	// New file must be written
	newContent, err := os.ReadFile(newFilePath)
	require.NoError(t, err)
	assert.Equal(t, "brand new content", string(newContent))
}
