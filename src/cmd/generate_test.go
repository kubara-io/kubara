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
	assert.Equal(t, "kubara generate [--all|--hubs HUB1,HUB2,...|--hub HUB3] [--terraform|--helm] [--catalog PATH_OR_OCI] [--catalog-overwrite]] [--dry-run]", command.UsageText)
	assert.Equal(t, "Renders Helm and Terraform templates from configured local or OCI catalogs for the specified hubs.\nBy default, it generates both template types and targets the hub in the current working directory.", command.Description)

	// Check that flags are added
	require.Len(t, command.Flags, 5)

	flagNames := make(map[string]bool)
	for _, flag := range command.Flags {
		flagNames[flag.Names()[0]] = true
	}

	assert.True(t, flagNames["terraform"])
	assert.True(t, flagNames["helm"])
	assert.True(t, flagNames["dry-run"])
	assert.True(t, flagNames["all"])
	assert.True(t, flagNames["hub"])
}

func TestGenerateCmd(t *testing.T) {

	tests := []struct {
		name        string
		flags       []string
		wantErr     bool
		errContains string
		cluster     *config.Cluster // overrides the default SKE test cluster when set
		skipConfig  bool
		skipEnv     bool
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
			name: "error with missing config file",
			flags: []string{
				"--dry-run",
			},
			skipConfig:  true,
			wantErr:     true,
			errContains: "load config",
		},
		{
			name: "error with missing env file",
			flags: []string{
				"--dry-run",
			},
			skipEnv:     true,
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

				// Provider selector folders are internal to catalog templates
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
					DNS:               config.DNS{Name: "example.com", Email: "admin@example.com"},
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
			t.Chdir(tempDir)
			require.NoError(t, os.Mkdir(filepath.Join(tempDir, ".git"), 0755))

			if !tt.skipConfig {
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
						DNS: config.DNS{
							Name:  "example.com",
							Email: "test" + "@" + "example.com",
						},
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
				testutil.CreateTestConfig(t, tempDir, cluster)
			}

			if !tt.skipEnv {
				testutil.CreateDefaultGenerateTestEnv(t, tempDir)
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
	t.Chdir(tempDir)

	testutil.CreateTestConfig(t, tempDir, config.Cluster{
		Name:    "no-provider-cluster",
		Stage:   "dev",
		Type:    "hub",
		DNSName: "test.example.com",
		Terraform: &config.Terraform{
			Provider:          "",
			ProjectID:         "00000000-0000-0000-0000-000000000000",
			KubernetesType:    "ske",
			KubernetesVersion: "1.28.0",
			DNS:               config.DNS{Name: "example.com", Email: "test" + "@" + "example.com"},
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
	args := []string{"kubara", "generate", "--terraform"}
	err := app.Run(context.Background(), args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing terraform configuration")
}

func TestGenerateCmd_MissingProviderUsesAllByDefault(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	helperCatalogPath := createHelperCatalog(t, tempDir)

	testutil.CreateTestConfig(t, tempDir, config.Cluster{
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
			DNS:               config.DNS{Name: "example.com", Email: "test" + "@" + "example.com"},
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
	args := []string{"kubara", "generate"}
	err := app.Run(context.Background(), args)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, "platform-components", "helpers", "readme.txt"))
	assert.NoFileExists(t, filepath.Join(tempDir, "platform-components", "terraform", "disabled.txt"))
}

func TestGenerateCmd_MissingTerraformUsesAllByDefault(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	helperCatalogPath := createHelperCatalog(t, tempDir)

	testutil.CreateTestConfig(t, tempDir, config.Cluster{
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
	// A user-managed file placed next to generated Terraform output must survive
	// a generate run; kubara never wipes per-cluster platform-configs directories.
	userTerraform := filepath.Join(tempDir, "platform-configs", "helm-only-cluster", "terraform", "extra.tf")
	require.NoError(t, os.MkdirAll(filepath.Dir(userTerraform), 0o750))
	require.NoError(t, os.WriteFile(userTerraform, []byte("user\n"), 0o600))

	app := CreateTestApp(NewGenerateCmd())
	args := []string{"kubara", "generate"}
	err := app.Run(context.Background(), args)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, "platform-components", "helpers", "readme.txt"))
	assert.NoFileExists(t, filepath.Join(tempDir, "platform-components", "terraform", "disabled.txt"))
	assert.FileExists(t, userTerraform)
}

func TestGenerateCmd_TerraformProviderNoneUsesAllByDefault(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	helperCatalogPath := createHelperCatalog(t, tempDir)

	testutil.CreateTestConfig(t, tempDir, config.Cluster{
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
	args := []string{"kubara", "generate"}
	err := app.Run(context.Background(), args)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(tempDir, "platform-components", "helpers", "readme.txt"))
	assert.NoFileExists(t, filepath.Join(tempDir, "platform-components", "terraform", "disabled.txt"))
}

func TestGenerateCmd_MissingTerraformFailsForTerraform(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)

	testutil.CreateTestConfig(t, tempDir, config.Cluster{
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
	args := []string{"kubara", "generate", "--terraform", "--dry-run"}
	err := app.Run(context.Background(), args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing terraform configuration")
}

func TestDisabledServicesDontGetWritten(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	services := testutil.CreateTestServices()
	serviceName := "cert-manager"
	certManager := services[serviceName]
	certManager.Status = "disabled"
	services[serviceName] = certManager

	testutil.CreateTestConfig(t, tempDir, config.Cluster{
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
	args := []string{"kubara", "generate"}
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

func TestGenerate_MultiSetup_Isolation(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	require.NoError(t, os.Mkdir(filepath.Join(tempDir, ".git"), 0755))

	// Setup A
	setupADir := filepath.Join(tempDir, "setups", "fleet-a")
	require.NoError(t, os.MkdirAll(setupADir, 0755))
	clusterA := config.Cluster{
		Name:    "hub-a",
		Stage:   "dev",
		Type:    "hub",
		DNSName: "a.example.com",
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				HTTPS: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
				},
			},
		},
		Services: service.Services{},
	}
	testutil.CreateTestConfig(t, setupADir, clusterA)
	testutil.CreateDefaultGenerateTestEnv(t, setupADir)

	// Setup B
	setupBDir := filepath.Join(tempDir, "setups", "fleet-b")
	require.NoError(t, os.MkdirAll(setupBDir, 0755))
	clusterB := config.Cluster{
		Name:    "hub-b",
		Stage:   "prod",
		Type:    "hub",
		DNSName: "b.example.com",
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				HTTPS: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
				},
			},
		},
		Services: service.Services{},
	}
	testutil.CreateTestConfig(t, setupBDir, clusterB)
	testutil.CreateDefaultGenerateTestEnv(t, setupBDir)

	// 1. Generate Fleet A using kubara generate --hub setups/fleet-a
	appA := CreateTestApp(NewGenerateCmd())
	argsA := []string{"kubara", "generate", "--hub", "setups/fleet-a"}
	err := appA.Run(context.Background(), argsA)
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(setupADir, "platform-components"))
	assert.DirExists(t, filepath.Join(setupADir, "platform-configs", "hub-a"))
	assert.NoDirExists(t, filepath.Join(setupBDir, "platform-components"))

	// Write a canary file into setupADir platform-components
	canaryFile := filepath.Join(setupADir, "platform-components", "canary.txt")
	require.NoError(t, os.WriteFile(canaryFile, []byte("fleet-a canary"), 0644))

	// 2. Generate Fleet B using kubara generate --hub setups/fleet-b
	appB := CreateTestApp(NewGenerateCmd())
	argsB := []string{"kubara", "generate", "--hub", "setups/fleet-b"}
	err = appB.Run(context.Background(), argsB)
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(setupBDir, "platform-components"))
	assert.DirExists(t, filepath.Join(setupBDir, "platform-configs", "hub-b"))

	// Canary file in fleet-a must still exist (fleet-a was not wiped by fleet-b generate!)
	assert.FileExists(t, canaryFile)
}

func TestGenerate_All_Workspaces(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	require.NoError(t, os.Mkdir(filepath.Join(tempDir, ".git"), 0755))

	// Setup 1
	setup1Dir := filepath.Join(tempDir, "setups", "fleet-1")
	require.NoError(t, os.MkdirAll(setup1Dir, 0755))
	cluster1 := config.Cluster{
		Name:    "hub-1",
		Stage:   "dev",
		Type:    "hub",
		DNSName: "1.example.com",
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				HTTPS: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
				},
			},
		},
		Services: service.Services{},
	}
	testutil.CreateTestConfig(t, setup1Dir, cluster1)
	testutil.CreateDefaultGenerateTestEnv(t, setup1Dir)

	// Setup 2
	setup2Dir := filepath.Join(tempDir, "setups", "fleet-2")
	require.NoError(t, os.MkdirAll(setup2Dir, 0755))
	cluster2 := config.Cluster{
		Name:    "hub-2",
		Stage:   "prod",
		Type:    "hub",
		DNSName: "2.example.com",
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				HTTPS: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
				},
			},
		},
		Services: service.Services{},
	}
	testutil.CreateTestConfig(t, setup2Dir, cluster2)
	testutil.CreateDefaultGenerateTestEnv(t, setup2Dir)

	// Generate --all from tempDir
	app := CreateTestApp(NewGenerateCmd())
	args := []string{"kubara", "generate", "--all"}
	err := app.Run(context.Background(), args)
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(setup1Dir, "platform-components"))
	assert.DirExists(t, filepath.Join(setup1Dir, "platform-configs", "hub-1"))
	assert.DirExists(t, filepath.Join(setup2Dir, "platform-components"))
	assert.DirExists(t, filepath.Join(setup2Dir, "platform-configs", "hub-2"))
}

func TestGenerate_Multiple_Hubs(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	require.NoError(t, os.Mkdir(filepath.Join(tempDir, ".git"), 0755))

	// Setup 1
	setup1Dir := filepath.Join(tempDir, "setups", "fleet-1")
	require.NoError(t, os.MkdirAll(setup1Dir, 0755))
	cluster1 := config.Cluster{
		Name:    "hub-1",
		Stage:   "dev",
		Type:    "hub",
		DNSName: "1.example.com",
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				HTTPS: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
				},
			},
		},
		Services: service.Services{},
	}
	testutil.CreateTestConfig(t, setup1Dir, cluster1)
	testutil.CreateDefaultGenerateTestEnv(t, setup1Dir)

	// Setup 2
	setup2Dir := filepath.Join(tempDir, "setups", "fleet-2")
	require.NoError(t, os.MkdirAll(setup2Dir, 0755))
	cluster2 := config.Cluster{
		Name:    "hub-2",
		Stage:   "prod",
		Type:    "hub",
		DNSName: "2.example.com",
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				HTTPS: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
				},
			},
		},
		Services: service.Services{},
	}
	testutil.CreateTestConfig(t, setup2Dir, cluster2)
	testutil.CreateDefaultGenerateTestEnv(t, setup2Dir)

	// Setup 3 (should NOT be generated)
	setup3Dir := filepath.Join(tempDir, "setups", "fleet-3")
	require.NoError(t, os.MkdirAll(setup3Dir, 0755))
	cluster3 := config.Cluster{
		Name:    "hub-3",
		Stage:   "staging",
		Type:    "hub",
		DNSName: "3.example.com",
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				HTTPS: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/repo", TargetRevision: "main"},
				},
			},
		},
		Services: service.Services{},
	}
	testutil.CreateTestConfig(t, setup3Dir, cluster3)
	testutil.CreateDefaultGenerateTestEnv(t, setup3Dir)

	tests := []struct {
		name              string
		args              []string
		wantErr           bool
		errContains       string
		expectedGenerated []string
		expectedSkipped   []string
	}{
		{
			name:              "comma-separated hubs",
			args:              []string{"kubara", "generate", "--hub", "setups/fleet-1,setups/fleet-2"},
			expectedGenerated: []string{setup1Dir, setup2Dir},
			expectedSkipped:   []string{setup3Dir},
		},
		{
			name:              "repeated --hub flags",
			args:              []string{"kubara", "generate", "--hub", "setups/fleet-1", "--hub", "setups/fleet-2"},
			expectedGenerated: []string{setup1Dir, setup2Dir},
			expectedSkipped:   []string{setup3Dir},
		},
		{
			name:        "both --all and --hub returns error",
			args:        []string{"kubara", "generate", "--all", "--hub", "setups/fleet-1"},
			wantErr:     true,
			errContains: "cannot specify both --all and --hub",
		},
		{
			name:        "empty hub string returns error",
			args:        []string{"kubara", "generate", "--hub", "  "},
			wantErr:     true,
			errContains: "no valid hub specified in --hub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up output directories before each run
			for _, dir := range []string{setup1Dir, setup2Dir, setup3Dir} {
				_ = os.RemoveAll(filepath.Join(dir, "platform-components"))
				_ = os.RemoveAll(filepath.Join(dir, "platform-configs"))
			}

			app := CreateTestApp(NewGenerateCmd())
			err := app.Run(context.Background(), tt.args)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			for _, dir := range tt.expectedGenerated {
				assert.DirExists(t, filepath.Join(dir, "platform-components"))
				assert.DirExists(t, filepath.Join(dir, "platform-configs"))
			}
			for _, dir := range tt.expectedSkipped {
				assert.NoDirExists(t, filepath.Join(dir, "platform-components"))
			}
		})
	}
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
