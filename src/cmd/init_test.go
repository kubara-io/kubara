package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	cmdtestutil "github.com/kubara-io/kubara/cmd/testutil"
	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	internaltestutil "github.com/kubara-io/kubara/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
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
	assert.Empty(t, flags.BootstrapCatalogFlag)
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
	assert.True(t, flagNames["bootstrap-catalog"])
}

func TestInitPersistsBootstrapCatalogOverride(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	bootstrapPath, generalPath, err := internaltestutil.CreateCatalogFixtures(filepath.Join(workDir, "catalogs"))
	require.NoError(t, err)
	envPath := cmdtestutil.CreateDefaultGenerateTestEnv(t, workDir)
	configPath := filepath.Join(workDir, "config.yaml")
	app := cmdtestutil.CreateTestAppWithFlags(NewGlobalFlags().CLIFlags(), NewInitCmd())

	err = app.Run(context.Background(), []string{
		"kubara",
		"--work-dir", workDir,
		"--config-file", configPath,
		"--env-file", envPath,
		"init",
		"--bootstrap-catalog", bootstrapPath,
		"--catalog", generalPath,
		"--catalog-overwrite",
		"--renovate=false",
	})
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var generated config.Config
	require.NoError(t, yaml.Unmarshal(data, &generated))
	require.NotNil(t, generated.BootstrapCatalog)
	assert.Equal(t, bootstrapPath, *generated.BootstrapCatalog)
	require.Len(t, generated.Clusters, 1)
	assert.Equal(t, []string{generalPath}, generated.Clusters[0].Catalogs)
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
