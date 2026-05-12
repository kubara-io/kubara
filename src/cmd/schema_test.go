package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaFlags(t *testing.T) {
	t.Parallel()

	flags := NewSchemaFlags()

	assert.Equal(t, "config.schema.json", flags.OutputFlag)
}

func TestNewSchemaCmd(t *testing.T) {
	t.Parallel()

	command := NewSchemaCmd()

	assert.Equal(t, "schema", command.Name)
	assert.Equal(t, "Generate JSON schema file for config structure", command.Usage)
	assert.Equal(t, "schema [--output] [--catalog <path> [--catalog-overwrite]]", command.UsageText)

	require.Len(t, command.Flags, 1)

	flagNames := make(map[string]bool)
	for _, flag := range command.Flags {
		flagNames[flag.Names()[0]] = true
	}

	assert.True(t, flagNames["output"])
}

func TestSchemaCmd(t *testing.T) {
	tests := []struct {
		name        string
		flags       []string
		wantErr     bool
		errContains string
		setup       func(t *testing.T, tempDir string)
		validate    func(t *testing.T, tempDir string)
	}{
		{
			name:    "successful schema generation with default output",
			flags:   []string{},
			wantErr: false,
			validate: func(t *testing.T, tempDir string) {
				schemaPath := filepath.Join(tempDir, "config.schema.json")
				assert.FileExists(t, schemaPath)

				data, err := os.ReadFile(schemaPath)
				require.NoError(t, err)

				var schemaDoc map[string]any
				err = json.Unmarshal(data, &schemaDoc)
				require.NoError(t, err)

				// Schema should have standard JSON Schema structure
				assert.Contains(t, schemaDoc, "$id")
				assert.Contains(t, schemaDoc, "properties")
				assert.Contains(t, schemaDoc, "$defs")
			},
		},
		{
			name: "successful schema generation with custom output path",
			flags: []string{
				"--output", "custom-schema.json",
			},
			wantErr: false,
			validate: func(t *testing.T, tempDir string) {
				schemaPath := filepath.Join(tempDir, "custom-schema.json")
				assert.FileExists(t, schemaPath)

				data, err := os.ReadFile(schemaPath)
				require.NoError(t, err)

				var schemaDoc map[string]any
				err = json.Unmarshal(data, &schemaDoc)
				require.NoError(t, err)
			},
		},
		{
			name: "successful schema generation with nested output path",
			flags: []string{
				"-o", "schemas/nested/config.schema.json",
			},
			wantErr: false,
			validate: func(t *testing.T, tempDir string) {
				// Directories should be created automatically
				schemaPath := filepath.Join(tempDir, "schemas", "nested", "config.schema.json")
				assert.FileExists(t, schemaPath)

				data, err := os.ReadFile(schemaPath)
				require.NoError(t, err)

				var schemaDoc map[string]any
				err = json.Unmarshal(data, &schemaDoc)
				require.NoError(t, err)
			},
		},
		{
			name: "schema output is pretty-printed",
			flags: []string{
				"--output", "pretty-schema.json",
			},
			wantErr: false,
			validate: func(t *testing.T, tempDir string) {
				schemaPath := filepath.Join(tempDir, "pretty-schema.json")
				data, err := os.ReadFile(schemaPath)
				require.NoError(t, err)

				content := string(data)
				assert.Contains(t, content, "\n")
				assert.Contains(t, content, "  ")
			},
		},
		{
			name: "catalog collision fails without force",
			flags: []string{
				"--catalog", "distribution",
			},
			setup: func(t *testing.T, tempDir string) {
				servicesDir := filepath.Join(tempDir, "distribution", "services")
				require.NoError(t, os.MkdirAll(servicesDir, 0750))
				require.NoError(t, os.WriteFile(filepath.Join(servicesDir, "argo-cd.yaml"), []byte(`
apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: argocd
spec:
  chartPath: custom-argo-cd
  status: enabled
`), 0644))
			},
			wantErr:     true,
			errContains: "already exists in built-in catalog",
		},
		{
			name: "catalog collision succeeds with catalog-overwrite",
			flags: []string{
				"--catalog", "distribution",
				"--catalog-overwrite",
			},
			setup: func(t *testing.T, tempDir string) {
				servicesDir := filepath.Join(tempDir, "distribution", "services")
				require.NoError(t, os.MkdirAll(servicesDir, 0750))
				require.NoError(t, os.WriteFile(filepath.Join(servicesDir, "argo-cd.yaml"), []byte(`
apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: argocd
spec:
  chartPath: custom-argo-cd
  status: enabled
`), 0644))
			},
			wantErr: false,
			validate: func(t *testing.T, tempDir string) {
				schemaPath := filepath.Join(tempDir, "config.schema.json")
				assert.FileExists(t, schemaPath)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			globalFlags := []string{
				"--work-dir", tempDir,
			}
			tt.flags = append(globalFlags, tt.flags...)

			if tt.setup != nil {
				tt.setup(t, tempDir)
			}

			app := createTestApp(NewSchemaCmd())

			// Run: kubara schema [flags]
			args := append([]string{"kubara", "schema"}, tt.flags...)

			err := app.Run(context.Background(), args)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, tempDir)
			}
		})
	}
}
