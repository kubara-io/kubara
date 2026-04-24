package cmd_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"kubara/internal/assets/config"
	"kubara/internal/assets/envmap"
	"kubara/cmd"
	"kubara/internal/assets/service"
	"kubara/internal/templates"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
	"go.yaml.in/yaml/v3"
)

func createTestServices() service.Services {
	return service.Services{
		"argo-cd":                 {Status: service.StatusEnabled},
		"cert-manager":            {Status: service.StatusEnabled, Config: service.Config{"clusterIssuer": map[string]any{"name": "letsencrypt-staging", "email": "admin@example.com", "server": "https://acme-staging-v02.api.letsencrypt.org/directory"}}},
		"external-dns":            {Status: service.StatusEnabled},
		"external-secrets":        {Status: service.StatusEnabled},
		"kube-prometheus-stack":   {Status: service.StatusEnabled, Storage: &service.Storage{ClassName: "standard-rwo"}},
		"traefik":                 {Status: service.StatusEnabled},
		"kyverno":                 {Status: service.StatusEnabled},
		"kyverno-policies":        {Status: service.StatusEnabled},
		"kyverno-policy-reporter": {Status: service.StatusEnabled},
		"loki":                    {Status: service.StatusEnabled, Storage: &service.Storage{ClassName: "standard-rwo"}},
		"homer-dashboard":         {Status: service.StatusEnabled},
		"oauth2-proxy":            {Status: service.StatusEnabled},
		"metrics-server":          {Status: service.StatusEnabled},
		"metallb":                 {Status: service.StatusEnabled},
		"longhorn":                {Status: service.StatusEnabled},
	}
}

func TestNewGenerateFlags(t *testing.T) {
	t.Parallel()

	flags := cmd.NewGenerateFlags()

	assert.False(t, flags.Terraform)
	assert.False(t, flags.Helm)
	assert.False(t, flags.DryRun)
	assert.Equal(t, templates.DefaultManagedCatalogPath, flags.ManagedCatalogPath)
	assert.Equal(t, templates.DefaultOverlayValuesPath, flags.OverlayValuesPath)
}

func TestNewGenerateCmd(t *testing.T) {
	t.Parallel()

	command := cmd.NewGenerateCmd()

	assert.Equal(t, "generate", command.Name)
	assert.Equal(t, "generates files from embedded templates and the config file; by default for both Helm and Terraform", command.Usage)
	assert.Equal(t, "generate [--terraform|--helm] [--managed-catalog <path> --overlay-values <path>] [--catalog <path> [--catalog-overwrite]] [--dry-run]", command.UsageText)
	assert.Equal(t, "generate reads config values and templates the embedded Helm and Terraform files.", command.Description)

	// Check that flags are added
	require.Len(t, command.Flags, 5)

	flagNames := make(map[string]bool)
	for _, flag := range command.Flags {
		flagNames[flag.Names()[0]] = true
	}

	assert.True(t, flagNames["terraform"])
	assert.True(t, flagNames["helm"])
	assert.True(t, flagNames["dry-run"])
	assert.True(t, flagNames["managed-catalog"])
	assert.True(t, flagNames["overlay-values"])
}

func TestGenerateCmd(t *testing.T) {

	tests := []struct {
		name        string
		flags       []string
		wantErr     bool
		errContains string
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
			errContains: "failed to load config",
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
				// Create managed catalog directory
				err := os.MkdirAll(filepath.Join(tempDir, "managed-service-catalog"), 0750)
				require.NoError(t, err)
			},
			validate: func(t *testing.T, tempDir string) {
				// Check that terraform files were generated
				terraformDir := filepath.Join(tempDir, "managed-service-catalog", "terraform")
				entries, err := os.ReadDir(terraformDir)
				require.NoError(t, err)
				assert.NotEmpty(t, entries)

				// Provider selector folders are internal to embedded templates
				// and must not leak into generated output paths.
				_, err = os.Stat(filepath.Join(terraformDir, "modules", "ske-cluster", "main.tf"))
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
				// Create managed catalog directory
				err := os.MkdirAll(filepath.Join(tempDir, "managed-service-catalog"), 0750)
				require.NoError(t, err)
			},
			validate: func(t *testing.T, tempDir string) {
				// Check that helm files were generated
				helmDir := filepath.Join(tempDir, "managed-service-catalog", "helm")
				entries, err := os.ReadDir(helmDir)
				require.NoError(t, err)
				assert.NotEmpty(t, entries)
			},
		},
		{
			name: "successful file generation with custom paths",
			flags: []string{
				"--terraform",
				"--managed-catalog", "custom-managed",
				"--overlay-values", "custom-overlay",
			},
			wantErr: false,
			setup: func(t *testing.T, tempDir string) {
				// Create custom directories
				err := os.MkdirAll(filepath.Join(tempDir, "custom-managed"), 0750)
				require.NoError(t, err)
				err = os.MkdirAll(filepath.Join(tempDir, "custom-overlay"), 0750)
				require.NoError(t, err)
			},
			validate: func(t *testing.T, tempDir string) {
				// Check that files were generated in custom paths
				terraformDir := filepath.Join(tempDir, "custom-managed", "terraform")
				entries, err := os.ReadDir(terraformDir)
				require.NoError(t, err)
				assert.NotEmpty(t, entries)
				_, err = os.Stat(filepath.Join(terraformDir, "modules", "ske-cluster", "main.tf"))
				require.NoError(t, err)
				_, err = os.Stat(filepath.Join(terraformDir, "providers"))
				assert.ErrorIs(t, err, os.ErrNotExist)

				// Check overlay files were generated with cluster name
				overlayDir := filepath.Join(tempDir, "custom-overlay")
				entries, err = os.ReadDir(overlayDir)
				require.NoError(t, err)
				assert.NotEmpty(t, entries)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tempDir := t.TempDir()

			// Create config file if not testing error case
			if !tt.wantErr || tt.errContains != "failed to load config" {
				configPath := createTestConfig(t, tempDir, config.Cluster{
					Name:             "test-cluster",
					Stage:            "dev",
					IngressClassName: "traefik",
					Type:             "controlplane",
					DNSName:          "test.example.com",
					Terraform: &config.Terraform{
						Provider:          "stackit",
						ProjectID:         "00000000-0000-0000-0000-000000000000",
						KubernetesType:    "ske",
						KubernetesVersion: "1.28.0",
						DNS: config.DNS{
							Name:  "example.com",
							Email: "admin@example.com",
						},
					},
					ArgoCD: config.ArgoCD{
						Repo: config.RepoProto{
							HTTPS: &config.RepoType{
								Customer: config.Repository{
									URL:            "https://github.com/example/customer",
									TargetRevision: "main",
								},
								Managed: config.Repository{
									URL:            "https://github.com/example/managed",
									TargetRevision: "main",
								},
							},
						},
					},
					Services: createTestServices(),
				})

				//dummy values
				envPath := createTestEnv(t, tempDir, envmap.EnvMap{
					ProjectName:                 "project-name",
					ProjectStage:                "project-stage",
					DockerconfigBase64:          "DockerConfig",
					ArgocdWizardAccountPassword: "wizardpassword",
					ArgocdGitHttpsUrl:           "https://example.com",
					ArgocdGitUsername:           "CoolCapybara",
					ArgocdGitPatOrPassword:      "password",
					ArgocdHelmRepoUrl:           "https://example.com",
					ArgocdHelmRepoUsername:      "CoolCapybara",
					ArgocdHelmRepoPassword:      "password",
					DomainName:                  "example.com",
				})

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
			app := createTestApp(cmd.NewGenerateCmd())

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

func TestGenerateCmd_MissingProviderUsesDefault(t *testing.T) {
	tempDir := t.TempDir()

	configPath := createTestConfig(t, tempDir, config.Cluster{
		Name:    "no-provider-cluster",
		Stage:   "dev",
		Type:    "controlplane",
		DNSName: "test.example.com",
		Terraform: &config.Terraform{
			Provider:          "",
			ProjectID:         "00000000-0000-0000-0000-000000000000",
			KubernetesType:    "ske",
			KubernetesVersion: "1.28.0",
			DNS:               config.DNS{Name: "example.com", Email: "admin@example.com"},
		},
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				HTTPS: &config.RepoType{
					Customer: config.Repository{URL: "https://github.com/example/customer", TargetRevision: "main"},
					Managed:  config.Repository{URL: "https://github.com/example/managed", TargetRevision: "main"},
				},
			},
		},
		Services: createTestServices(),
	})

	//dummy values
	createTestEnv(t, tempDir, envmap.EnvMap{
		ProjectName:                 "project-name",
		ProjectStage:                "project-stage",
		DockerconfigBase64:          "DockerConfig",
		ArgocdWizardAccountPassword: "wizardpassword",
		ArgocdGitHttpsUrl:           "https://example.com",
		ArgocdGitUsername:           "CoolCapybara",
		ArgocdGitPatOrPassword:      "password",
		ArgocdHelmRepoUrl:           "https://example.com",
		ArgocdHelmRepoUsername:      "CoolCapybara",
		ArgocdHelmRepoPassword:      "password",
		DomainName:                  "example.com",
	})

	app := createTestApp(cmd.NewGenerateCmd())
	args := []string{"kubara", "--config-file", configPath, "--work-dir", tempDir, "generate", "--terraform"}
	err := app.Run(context.Background(), args)
	require.NoError(t, err)

	terraformDir := filepath.Join(tempDir, "managed-service-catalog", "terraform")
	entries, err := os.ReadDir(terraformDir)
	require.NoError(t, err)
	assert.NotEmpty(t, entries)

	// Provider ske-cluster directory and main.tf exists
	// as stackit is the default provider when none is specified.
	_, err = os.Stat(filepath.Join(terraformDir, "modules", "ske-cluster", "main.tf"))
	require.NoError(t, err)

}

func TestGenerateCmd_PlaceholderProviderFailsWithHint(t *testing.T) {
	tempDir := t.TempDir()

	configPath := createTestConfig(t, tempDir, config.Cluster{
		Name:    "placeholder-provider-cluster",
		Stage:   "dev",
		Type:    "controlplane",
		DNSName: "test.example.com",
		Terraform: &config.Terraform{
			Provider:          "<provider>",
			ProjectID:         "00000000-0000-0000-0000-000000000000",
			KubernetesType:    "ske",
			KubernetesVersion: "1.28.0",
			DNS:               config.DNS{Name: "example.com", Email: "admin@example.com"},
		},
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				HTTPS: &config.RepoType{
					Customer: config.Repository{URL: "https://github.com/example/customer", TargetRevision: "main"},
					Managed:  config.Repository{URL: "https://github.com/example/managed", TargetRevision: "main"},
				},
			},
		},
		Services: createTestServices(),
	})

	app := createTestApp(cmd.NewGenerateCmd())

	//dummy values
	createTestEnv(t, tempDir, envmap.EnvMap{
		ProjectName:                 "project-name",
		ProjectStage:                "project-stage",
		DockerconfigBase64:          "DockerConfig",
		ArgocdWizardAccountPassword: "wizardpassword",
		ArgocdGitHttpsUrl:           "https://example.com",
		ArgocdGitUsername:           "CoolCapybara",
		ArgocdGitPatOrPassword:      "password",
		ArgocdHelmRepoUrl:           "https://example.com",
		ArgocdHelmRepoUsername:      "CoolCapybara",
		ArgocdHelmRepoPassword:      "password",
		DomainName:                  "example.com",
	})

	args := []string{"kubara", "--config-file", configPath, "--work-dir", tempDir, "generate", "--terraform", "--dry-run"}
	err := app.Run(context.Background(), args)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "placeholder provider")
	assert.Contains(t, err.Error(), "supported providers: stackit")
}

// Helper function

func createTestConfig(t *testing.T, dir string, clusters ...config.Cluster) string {
	t.Helper()

	configPath := filepath.Join(dir, "config.yaml")

	cfg := config.Config{Clusters: clusters}

	// Convert to YAML
	yamlData, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	err = os.WriteFile(configPath, yamlData, 0644)
	require.NoError(t, err)

	return configPath
}

// createTestEnv writes an envMap to the file system
// It returns the file path
// Takes a directory and an EnvMap and validates the envMap before writing it
func createTestEnv(t *testing.T, dir string, env envmap.EnvMap) string {
	envPath := filepath.Join(dir, ".env")

	manager := envmap.NewEnvMapManager(envPath, ".", "")
	manager.SetEnvMap(env)
	err := manager.ValidateAndSaveToFile(envPath)

	require.NoError(t, err)

	return envPath
}

func createTestApp(commands ...*cli.Command) *cli.Command {
	return &cli.Command{
		Name:     "kubara",
		Commands: commands,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config-file",
				Usage: "Path to the configuration file",
				Value: "config.yaml",
			},
			&cli.StringFlag{
				Name:  "work-dir",
				Usage: "Working directory",
				Value: ".",
			},
			&cli.StringFlag{
				Name:  "env-file",
				Usage: "Path to the .env file",
				Value: ".env",
			},
			&cli.StringFlag{
				Name:  "catalog",
				Usage: "Path to external ServiceDefinition catalog directory.",
				Value: "",
			},
			&cli.BoolFlag{
				Name:  "catalog-overwrite",
				Usage: "Allow external service definitions from --catalog to overwrite built-in definitions on name collisions.",
				Value: false,
			},
		},
	}
}
