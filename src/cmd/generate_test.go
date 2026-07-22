package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kubara-io/kubara/cmd/testutil"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func createHelperCatalog(t *testing.T, root string) string {
	t.Helper()

	catalogPath := filepath.Join(root, "helper-catalog")
	require.NoError(t, os.MkdirAll(filepath.Join(catalogPath, "services"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(catalogPath, "platform-components", "helpers"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(catalogPath, "platform-components", "terraform"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(catalogPath, "Catalog.yaml"), []byte(`apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: helper
spec:
  version: 1.0.0
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(catalogPath, "services", "cert-manager.yaml"), []byte(`apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: cert-manager
spec:
  chartPath: cert-manager
  status: enabled
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(catalogPath, "platform-components", "helpers", "readme.txt"), []byte("helper asset\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(catalogPath, "platform-components", "terraform", "disabled.txt"), []byte("terraform asset\n"), 0o600))

	return catalogPath
}

func TestNewGenerateFlags(t *testing.T) {
	t.Parallel()

	flags := NewGenerateFlags()

	assert.False(t, flags.Terraform)
	assert.False(t, flags.Helm)
	assert.False(t, flags.DryRun)
}

func TestNewGenerateCmd(t *testing.T) {
	t.Parallel()

	command := NewGenerateCmd()

	assert.Equal(t, "generate", command.Name)
	assert.Equal(t, "Generate files from catalog templates", command.Usage)
	assert.Equal(t, "kubara generate [--terraform|--helm] [--catalog PATH_OR_OCI [--catalog-overwrite]] [--dry-run]", command.UsageText)
	assert.Equal(t, "Renders embedded Helm and Terraform templates using values from the config file. By default, it generates both template types.", command.Description)

	// Check that flags are added
	require.Len(t, command.Flags, 3)

	flagNames := make(map[string]bool)
	for _, flag := range command.Flags {
		flagNames[flag.Names()[0]] = true
	}

	assert.True(t, flagNames["terraform"])
	assert.True(t, flagNames["helm"])
	assert.True(t, flagNames["dry-run"])
}

func TestGenerateCmd(t *testing.T) {

	tests := []struct {
		name        string
		flags       []string
		wantErr     bool
		errContains string
		cluster     *config.Cluster // overrides the default SKE test cluster when set
		setup       func(t *testing.T, tempDir string)
		validate    func(t *testing.T, tempDir string)
	}{
		{
			name: "successful terraform dry run",
			flags: []string{
				"--terraform",
				"--dry-run",
			},
			wantErr: false,
		},
		{
			name: "successful helm dry run",
			flags: []string{
				"--helm",
				"--dry-run",
			},
			wantErr: false,
		},
		{
			name: "successful all types dry run",
			flags: []string{
				"--dry-run",
			},
			wantErr: false,
		},
		{
			name: "error with non-existent config file",
			flags: []string{
				"--config-file", "/non/existent/config.yaml",
				"--dry-run",
			},
			wantErr:     true,
			errContains: "load config",
		},
		{
			name: "error with non-existent env file",
			flags: []string{
				"--env-file", "/non/existent/.env",
				"--dry-run",
			},
			wantErr:     true,
			errContains: "Vars not set",
		},
		{
			name: "successful terraform file generation",
			flags: []string{
				"--terraform",
			},
			wantErr: false,
			setup: func(t *testing.T, tempDir string) {
				// Create platform-components directory
				err := os.MkdirAll(filepath.Join(tempDir, "platform-components"), 0750)
				require.NoError(t, err)
			},
			validate: func(t *testing.T, tempDir string) {
				// Check that terraform files were generated
				terraformDir := filepath.Join(tempDir, "platform-components", "terraform")
				entries, err := os.ReadDir(terraformDir)
				require.NoError(t, err)
				assert.NotEmpty(t, entries)

				// Provider selector folders are internal to embedded templates
				// and must not leak into generated output paths.
				_, err = os.Stat(filepath.Join(terraformDir, "stackit", "modules", "ske-cluster", "main.tf"))
				require.NoError(t, err)
				_, err = os.Stat(filepath.Join(terraformDir, "providers"))
				assert.ErrorIs(t, err, os.ErrNotExist)
			},
		},
		{
			name: "successful helm file generation",
			flags: []string{
				"--helm",
			},
			wantErr: false,
			setup: func(t *testing.T, tempDir string) {
				// Create platform-components directory
				err := os.MkdirAll(filepath.Join(tempDir, "platform-components"), 0750)
				require.NoError(t, err)
			},
			validate: func(t *testing.T, tempDir string) {
				// Check that helm files were generated
				helmDir := filepath.Join(tempDir, "platform-components", "helm")
				entries, err := os.ReadDir(helmDir)
				require.NoError(t, err)
				assert.NotEmpty(t, entries)
			},
		},
		{
			name:    "successful edge terraform file generation",
			flags:   []string{"--terraform"},
			wantErr: false,
			cluster: &config.Cluster{
				Name:    "edge-cluster",
				Stage:   "dev",
				Type:    "hub",
				DNSName: "edge.example.com",
				Terraform: &config.Terraform{
					Provider:          "stackit",
					ProjectID:         "00000000-0000-0000-0000-000000000000",
					KubernetesType:    "edge",
					KubernetesVersion: "1.34.0",
					DNSContactEmail:   "admin@example.com",
				},
				ArgoCD: config.ArgoCD{
					Repo: config.RepoProto{
						Git: &config.RepoType{
							Configs:    config.Repository{URL: "https://github.com/example/configs", TargetRevision: "main"},
							Components: config.Repository{URL: "https://github.com/example/components", TargetRevision: "main"},
						},
					},
				},
				Services: service.Services{},
			},
			validate: func(t *testing.T, tempDir string) {
				// Edge renders the example infrastructure under the cluster name.
				// Assert the artifact set is produced, not its rendered content.
				infrastructureDir := filepath.Join(tempDir, "platform-configs", "edge-cluster", "terraform", "infrastructure")

				entries, err := os.ReadDir(infrastructureDir)
				require.NoError(t, err)
				assert.NotEmpty(t, entries)

				for _, name := range []string{"main.tf", "outputs.tf", "variables.tf", "env.auto.tfvars"} {
					_, statErr := os.Stat(filepath.Join(infrastructureDir, name))
					require.NoErrorf(t, statErr, "expected generated edge artifact %q", name)
				}

				// Provider selector folders does not exist anymore
				_, err = os.Stat(filepath.Join(tempDir, "platform-configs", "terraform", "providers"))
				assert.ErrorIs(t, err, os.ErrNotExist)

				// Provider folders must not leak into output paths.
				_, err = os.Stat(filepath.Join(tempDir, "platform-configs", "terraform", "stackit"))
				assert.ErrorIs(t, err, os.ErrNotExist)
				_, err = os.Stat(filepath.Join(tempDir, "platform-configs", "edge-cluster", "terraform", "stackit"))
				assert.ErrorIs(t, err, os.ErrNotExist)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tempDir := t.TempDir()

			// Create config file if not testing error case
			if !tt.wantErr || tt.errContains != "load config" {
				cluster := config.Cluster{
					Name:             "test-cluster",
					Stage:            "dev",
					IngressClassName: "traefik",
					Type:             "hub",
					DNSName:          "test.example.com",
					Terraform: &config.Terraform{
						Provider:          "stackit",
						ProjectID:         "00000000-0000-0000-0000-000000000000",
						KubernetesType:    "ske",
						KubernetesVersion: "1.28.0",
						DNSContactEmail:   "admin@example.com",
					},
					ArgoCD: config.ArgoCD{
						Repo: config.RepoProto{
							Git: &config.RepoType{
								Configs: config.Repository{
									URL:            "https://github.com/example/configs",
									TargetRevision: "main",
								},
								Components: config.Repository{
									URL:            "https://github.com/example/components",
									TargetRevision: "main",
								},
							},
						},
					},
					Services: service.Services{},
				}
				if tt.cluster != nil {
					cluster = *tt.cluster
				}
				configPath := testutil.CreateTestConfig(t, tempDir, cluster)

				//dummy values
				envPath := testutil.CreateDefaultGenerateTestEnv(t, tempDir)

				// Add global flags
				globalFlags := []string{
					"--config-file", configPath,
					"--work-dir", tempDir,
					"--env-file", envPath,
				}
				tt.flags = append(globalFlags, tt.flags...)
			}

			if tt.setup != nil {
				tt.setup(t, tempDir)
			}

			// Create app with generate command and global flags
			app := CreateTestApp(NewGenerateCmd())

			// Run: kubara generate [flags]
			args := append([]string{"kubara", "generate"}, tt.flags...)

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

func TestGenerateCmd_MissingProviderFailsForTerraform(t *testing.T) {
	tempDir := t.TempDir()

	configPath := testutil.CreateTestConfig(t, tempDir, config.Cluster{
		Name:    "no-provider-cluster",
		Stage:   "dev",
		Type:    "hub",
		DNSName: "test.example.com",
		Terraform: &config.Terraform{
			Provider:          "",
			ProjectID:         "00000000-0000-0000-0000-000000000000",
			KubernetesType:    "ske",
			KubernetesVersion: "1.28.0",
			DNSContactEmail:   "admin@example.com",
		},
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				Git: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/configs", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/components", TargetRevision: "main"},
				},
			},
		},
		Services: service.Services{},
	})

	//dummy values
	testutil.CreateDefaultGenerateTestEnv(t, tempDir)

	app := CreateTestApp(NewGenerateCmd())
	args := []string{"kubara", "--config-file", configPath, "--work-dir", tempDir, "generate", "--terraform"}
	err := app.Run(context.Background(), args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing terraform configuration")
}

func TestGenerateCmd_MissingProviderUsesAllByDefault(t *testing.T) {
	tempDir := t.TempDir()
	helperCatalogPath := createHelperCatalog(t, tempDir)

	configPath := testutil.CreateTestConfig(t, tempDir, config.Cluster{
		Name:     "no-provider-cluster",
		Stage:    "dev",
		Type:     "hub",
		DNSName:  "test.example.com",
		Catalogs: []string{helperCatalogPath},
		Terraform: &config.Terraform{
			Provider:          "",
			ProjectID:         "00000000-0000-0000-0000-000000000000",
			KubernetesType:    "ske",
			KubernetesVersion: "1.28.0",
			DNSContactEmail:   "admin@example.com",
		},
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				Git: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/configs", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/components", TargetRevision: "main"},
				},
			},
		},
		Services: service.Services{},
	})

	//dummy values
	testutil.CreateDefaultGenerateTestEnv(t, tempDir)

	app := CreateTestApp(NewGenerateCmd())
	args := []string{"kubara", "--config-file", configPath, "--work-dir", tempDir, "generate"}
	err := app.Run(context.Background(), args)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, "platform-components", "helpers", "readme.txt"))
	assert.NoFileExists(t, filepath.Join(tempDir, "platform-components", "terraform", "disabled.txt"))
}

func TestGenerateCmd_MissingTerraformUsesAllByDefault(t *testing.T) {
	tempDir := t.TempDir()
	helperCatalogPath := createHelperCatalog(t, tempDir)

	configPath := testutil.CreateTestConfig(t, tempDir, config.Cluster{
		Name:     "helm-only-cluster",
		Stage:    "dev",
		Type:     "hub",
		DNSName:  "test.example.com",
		Catalogs: []string{helperCatalogPath},
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				Git: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/configs", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/components", TargetRevision: "main"},
				},
			},
		},
		Services: service.Services{},
	})

	//dummy values
	testutil.CreateDefaultGenerateTestEnv(t, tempDir)
	staleTerraform := filepath.Join(tempDir, "platform-configs", "helm-only-cluster", "terraform", "stale.tf")
	require.NoError(t, os.MkdirAll(filepath.Dir(staleTerraform), 0o750))
	require.NoError(t, os.WriteFile(staleTerraform, []byte("stale\n"), 0o600))

	app := CreateTestApp(NewGenerateCmd())
	args := []string{"kubara", "--config-file", configPath, "--work-dir", tempDir, "generate"}
	err := app.Run(context.Background(), args)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, "platform-components", "helpers", "readme.txt"))
	assert.NoFileExists(t, filepath.Join(tempDir, "platform-components", "terraform", "disabled.txt"))
	assert.NoFileExists(t, staleTerraform)
}

func TestGenerateCmd_TerraformProviderNoneUsesAllByDefault(t *testing.T) {
	tempDir := t.TempDir()
	helperCatalogPath := createHelperCatalog(t, tempDir)

	configPath := testutil.CreateTestConfig(t, tempDir, config.Cluster{
		Name:     "provider-none-cluster",
		Stage:    "dev",
		Type:     "hub",
		DNSName:  "test.example.com",
		Catalogs: []string{helperCatalogPath},
		Terraform: &config.Terraform{
			Provider: config.TerraformProviderNone,
		},
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				Git: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/configs", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/components", TargetRevision: "main"},
				},
			},
		},
		Services: service.Services{},
	})

	testutil.CreateDefaultGenerateTestEnv(t, tempDir)

	app := CreateTestApp(NewGenerateCmd())
	args := []string{"kubara", "--config-file", configPath, "--work-dir", tempDir, "generate"}
	err := app.Run(context.Background(), args)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, "platform-components", "helpers", "readme.txt"))
	assert.NoFileExists(t, filepath.Join(tempDir, "platform-components", "terraform", "disabled.txt"))
}

func TestGenerateCmd_MissingTerraformFailsForTerraform(t *testing.T) {
	tempDir := t.TempDir()

	configPath := testutil.CreateTestConfig(t, tempDir, config.Cluster{
		Name:    "missing-terraform-cluster",
		Stage:   "dev",
		Type:    "hub",
		DNSName: "test.example.com",
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				Git: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/configs", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/components", TargetRevision: "main"},
				},
			},
		},
		Services: testutil.CreateTestServices(),
	})

	//dummy values
	testutil.CreateDefaultGenerateTestEnv(t, tempDir)

	app := CreateTestApp(NewGenerateCmd())
	args := []string{"kubara", "--config-file", configPath, "--work-dir", tempDir, "generate", "--terraform", "--dry-run"}
	err := app.Run(context.Background(), args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing terraform configuration")
}

func TestDisabledServicesDontGetWritten(t *testing.T) {
	tempDir := t.TempDir()
	services := testutil.CreateTestServices()
	serviceName := "cert-manager"
	certManager := services[serviceName]
	certManager.Status = "disabled"
	services[serviceName] = certManager

	configPath := testutil.CreateTestConfig(t, tempDir, config.Cluster{
		Name:    "missing-terraform-cluster",
		Stage:   "dev",
		Type:    "hub",
		DNSName: "test.example.com",
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				Git: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/configs", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/components", TargetRevision: "main"},
				},
			},
		},
		Services: services,
	})
	//dummy values
	testutil.CreateDefaultGenerateTestEnv(t, tempDir)

	app := CreateTestApp(NewGenerateCmd())
	args := []string{"kubara", "--config-file", configPath, "--work-dir", tempDir, "generate"}
	err := app.Run(context.Background(), args)
	require.NoError(t, err)

	helmDir := filepath.Join(tempDir, "platform-components", "helm")
	entries, err := os.ReadDir(helmDir)
	names := make([]string, 0)
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	require.NoError(t, err)
	assert.NotContains(t, names, serviceName)
}

// Helper function

func CreateTestApp(commands ...*cli.Command) *cli.Command {
	globalFlags := NewGlobalFlags()

	return &cli.Command{
		Name:     "kubara",
		Commands: commands,
		Flags:    globalFlags.CLIFlags(),
	}
}
