package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInitFlags(t *testing.T) {
	t.Parallel()

	flags := NewInitFlags()

	assert.False(t, flags.PrepFlag)
	assert.False(t, flags.ForceFlag)
	assert.False(t, flags.LocalFlag)
	assert.True(t, flags.RenovateFlag)
	assert.Equal(t, ".env", flags.EnvFileFlag)
	assert.Equal(t, "KUBARA_", flags.EnvPrefixFlag)
}

func TestNewInitCmd(t *testing.T) {
	t.Parallel()

	command := NewInitCmd()
	flagNames := make(map[string]bool, len(command.Flags))
	for _, flag := range command.Flags {
		flagNames[flag.Names()[0]] = true
	}

	assert.Equal(t, "init", command.Name)
	assert.True(t, flagNames["prep"])
	assert.True(t, flagNames["overwrite"])
	assert.True(t, flagNames["local"])
	assert.True(t, flagNames["renovate"])
	assert.True(t, flagNames["envVarPrefix"])
}

func TestEnsureRenovateConfig(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	configPath := filepath.Join(workDir, "clusters", "prod", "config.yaml")
	store := config.NewConfigStore(workDir, configPath, catalog.LoadOptions{})
	options := &InitOptions{cwd: workDir, renovate: true}

	require.NoError(t, options.ensureRenovateConfig(store))

	renovatePath := filepath.Join(workDir, "renovate.json")
	assert.FileExists(t, renovatePath)
	assert.NoFileExists(t, filepath.Join(filepath.Dir(configPath), "renovate.json"))

	content, err := os.ReadFile(renovatePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), `(?<depName>`)
	assert.NotContains(t, string(content), `\u003c`)

	var generated renovateConfig
	require.NoError(t, json.Unmarshal(content, &generated))
	require.Len(t, generated.CustomManagers, 1)
	manager := generated.CustomManagers[0]
	assert.Equal(t, []string{`/^clusters\/prod\/config\.yaml$/`}, manager.ManagerFilePatterns)
	assert.Equal(t, "docker", manager.DatasourceTemplate)
	assert.Equal(t, `regex:^(?<major>0|[1-9]\d*)\.(?<minor>0|[1-9]\d*)\.(?<patch>0|[1-9]\d*)$`, manager.VersioningTemplate)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(content, &raw))
	assert.NotContains(t, raw, "enabledManagers")
	assert.NotContains(t, raw, "packageRules")
	customManagers := raw["customManagers"].([]any)
	customManager := customManagers[0].(map[string]any)
	assert.Contains(t, customManager, "managerFilePatterns")
	assert.NotContains(t, customManager, "fileMatch")

	matchPattern := strings.ReplaceAll(manager.MatchStrings[0], `\/`, "/")
	matchRegexp := regexp.MustCompile(matchPattern)
	match := matchRegexp.FindStringSubmatch("catalog: oci://registry.example.com:5000/platform/security:1.2.3")
	require.NotNil(t, match)
	assert.Equal(t, "registry.example.com:5000/platform/security", match[matchRegexp.SubexpIndex("depName")])
	assert.Equal(t, "1.2.3", match[matchRegexp.SubexpIndex("currentValue")])
	assert.Nil(t, matchRegexp.FindStringSubmatch("catalog: oci://registry.example.com/platform/security@sha256:abcdef"))
	assert.Nil(t, matchRegexp.FindStringSubmatch("catalog: oci://registry.example.com/platform/security:1.2.3@sha256:abcdef"))

	require.NoError(t, options.ensureRenovateConfig(store))
}

func TestRenovateConfigContainsKubaraManager(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	renovatePath := filepath.Join(workDir, "renovate.jsonc")
	require.NoError(t, os.WriteFile(renovatePath, []byte(`{
  // Generated config may contain comments.
  "customManagers": [{"description": "Update kubara catalog OCI references"}]
}
`), 0o600))

	contains, err := renovateConfigContainsKubaraManager(renovatePath)
	require.NoError(t, err)
	assert.True(t, contains)
}

func TestEnsureRenovateConfigDisabled(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	store := config.NewConfigStore(workDir, filepath.Join(workDir, "config.yaml"), catalog.LoadOptions{})
	options := &InitOptions{cwd: workDir, renovate: false}

	require.NoError(t, options.ensureRenovateConfig(store))
	assert.NoFileExists(t, filepath.Join(workDir, "renovate.json"))
}

func TestEnsureRenovateConfigPreservesExistingConfig(t *testing.T) {
	t.Parallel()

	renovateFiles := []string{
		"renovate.json",
		"renovate.jsonc",
		"renovate.json5",
		".github/renovate.json",
		".github/renovate.jsonc",
		".github/renovate.json5",
		".gitlab/renovate.json",
		".gitlab/renovate.jsonc",
		".gitlab/renovate.json5",
		".renovaterc",
		".renovaterc.json",
		".renovaterc.jsonc",
		".renovaterc.json5",
	}

	for _, renovateFile := range renovateFiles {
		renovateFile := renovateFile
		t.Run(renovateFile, func(t *testing.T) {
			t.Parallel()

			workDir := t.TempDir()
			existingPath := filepath.Join(workDir, renovateFile)
			require.NoError(t, os.MkdirAll(filepath.Dir(existingPath), 0o750))
			require.NoError(t, os.WriteFile(existingPath, []byte("existing\n"), 0o600))

			store := config.NewConfigStore(workDir, filepath.Join(workDir, "config.yaml"), catalog.LoadOptions{})
			options := &InitOptions{cwd: workDir, renovate: true}
			require.NoError(t, options.ensureRenovateConfig(store))

			content, err := os.ReadFile(existingPath)
			require.NoError(t, err)
			assert.Equal(t, "existing\n", string(content))
			if renovateFile != "renovate.json" {
				assert.NoFileExists(t, filepath.Join(workDir, "renovate.json"))
			}
		})
	}
}

func TestRunNormalModeDoesNotCreateRenovateConfigOnValidationError(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	store := config.NewConfigStore(workDir, filepath.Join(workDir, "config.yaml"), catalog.LoadOptions{})
	options := &InitOptions{cwd: workDir, renovate: true}

	err := options.runNormalMode(nil, store, errors.New("invalid environment"))
	require.ErrorContains(t, err, "validate env")
	assert.NoFileExists(t, filepath.Join(workDir, "renovate.json"))
}
