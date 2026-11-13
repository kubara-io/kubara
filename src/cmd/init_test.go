package cmd

import (
	"context"
	"encoding/json"
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

func TestInitWarnsIfConfigAlreadyExistsAndCreatesMissingFiles(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	configPath := filepath.Join(workDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("existing: true\n"), 0o600))

	app := cmdtestutil.CreateTestAppWithFlags(NewGlobalFlags().CLIFlags(), NewInitCmd())
	err := app.Run(context.Background(), []string{
		"kubara",
		"--work-dir", workDir,
		"--config-file", configPath,
		"init",
	})
	require.NoError(t, err)

	// Existing config file must not be overwritten
	configBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "existing: true\n", string(configBytes))

	// Missing files should be created
	envExamplePath := filepath.Join(workDir, "env.example")
	assert.FileExists(t, envExamplePath)

	renovatePath := filepath.Join(workDir, "renovate.json")
	assert.FileExists(t, renovatePath)

	gitignorePath := filepath.Join(workDir, ".gitignore")
	assert.FileExists(t, gitignorePath)

	// Running init a second time when all files exist should also succeed without error
	err = app.Run(context.Background(), []string{
		"kubara",
		"--work-dir", workDir,
		"--config-file", configPath,
		"init",
	})
	require.NoError(t, err)
}

func TestInitLocalModeScaffoldsLocalCluster(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	configPath := filepath.Join(workDir, "config.yaml")
	app := cmdtestutil.CreateTestAppWithFlags(NewGlobalFlags().CLIFlags(), NewInitCmd())

	err := app.Run(context.Background(), []string{
		"kubara",
		"--work-dir", workDir,
		"--config-file", configPath,
		"init",
		"--local",
		"--renovate=false",
	})
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var generated struct {
		Spec config.Config `json:"spec"`
	}
	require.NoError(t, yaml.Unmarshal(data, &generated))
	require.Len(t, generated.Spec.Clusters, 1)
	assert.Equal(t, "test-cluster", generated.Spec.Clusters[0].Name)
	assert.Equal(t, "local", generated.Spec.Clusters[0].Stage)
	assert.Equal(t, "test-cluster-local.traefik.me", generated.Spec.Clusters[0].DNSName)
	assert.Nil(t, generated.Spec.Clusters[0].Terraform)

	// Verify .gitignore is created and contains .env and env.example
	gitignorePath := filepath.Join(workDir, ".gitignore")
	assert.FileExists(t, gitignorePath)
	gitignoreBytes, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)
	assert.Contains(t, string(gitignoreBytes), ".env")
	assert.Contains(t, string(gitignoreBytes), "env.example")

	// Verify env.example is generated
	envExamplePath := filepath.Join(workDir, "env.example")
	assert.FileExists(t, envExamplePath)
	envExampleBytes, err := os.ReadFile(envExamplePath)
	require.NoError(t, err)
	assert.Contains(t, string(envExampleBytes), "GITOPS_USERNAME")
	assert.Contains(t, string(envExampleBytes), "ARGOCD_PASSWORD")
}

func TestInitPersistsBootstrapCatalogOverride(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	bootstrapPath, generalPath, err := internaltestutil.CreateCatalogFixtures(filepath.Join(workDir, "catalogs"))
	require.NoError(t, err)
	configPath := filepath.Join(workDir, "config.yaml")
	app := cmdtestutil.CreateTestAppWithFlags(NewGlobalFlags().CLIFlags(), NewInitCmd())

	err = app.Run(context.Background(), []string{
		"kubara",
		"--work-dir", workDir,
		"--config-file", configPath,
		"init",
		"--bootstrap-catalog", bootstrapPath,
		"--catalog", generalPath,
		"--catalog-overwrite",
		"--renovate=false",
	})
	require.NoError(t, err)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var generated struct {
		Spec config.Config `json:"spec"`
	}
	require.NoError(t, yaml.Unmarshal(data, &generated))
	require.NotNil(t, generated.Spec.BootstrapCatalog)
	assert.Equal(t, bootstrapPath, *generated.Spec.BootstrapCatalog)
	require.Len(t, generated.Spec.Clusters, 1)
	assert.Equal(t, []string{generalPath}, generated.Spec.Clusters[0].Catalogs)
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

func TestInitPreservesExistingEnvExample(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	configPath := filepath.Join(workDir, "config.yaml")
	envExamplePath := filepath.Join(workDir, "env.example")
	require.NoError(t, os.WriteFile(envExamplePath, []byte("CUSTOM_CONTENT=1\n"), 0o600))

	app := cmdtestutil.CreateTestAppWithFlags(NewGlobalFlags().CLIFlags(), NewInitCmd())
	err := app.Run(context.Background(), []string{
		"kubara",
		"--work-dir", workDir,
		"--config-file", configPath,
		"init",
		"--renovate=false",
	})
	require.NoError(t, err)

	content, err := os.ReadFile(envExamplePath)
	require.NoError(t, err)
	assert.Equal(t, "CUSTOM_CONTENT=1\n", string(content))
}
