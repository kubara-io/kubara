package cmd_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"kubara/assets/config"
	"kubara/cmd"
	"kubara/templates"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

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
	assert.Equal(t, "generates files from embedded templates and config.", command.Usage)
	assert.Equal(t, "generate [--terraform|--helm] [--managed-catalog <path> --overlay-values <path>] [--dry-run]", command.UsageText)
	assert.Equal(t, "generate reads config values and templates the embedded helm and terraform files.", command.Description)

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

				// Check overlay files were generated with cluster name
				overlayDir := filepath.Join(tempDir, "custom-overlay")
				entries, err = os.ReadDir(overlayDir)
				require.NoError(t, err)
				assert.NotEmpty(t, entries)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {

			tempDir := t.TempDir()

			// Create config file if not testing error case
			if !tt.wantErr || tt.errContains != "failed to load config" {
				configPath := createTestConfig(t, tempDir, config.Cluster{
					Name:      "test-cluster",
					Stage:     "dev",
					ProjectID: "12345678-1234-1234-1234-123456789012",
					Type:      "controlplane",
					DNSName:   "test.example.com",
					Terraform: &config.Terraform{
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
					Services: config.Services{
						Argocd:              config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						CertManager:         config.CertManagerService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}, ClusterIssuer: config.ClusterIssuer{Name: "letsencrypt-staging", Email: "admin@example.com", Server: "https://acme-staging-v02.api.letsencrypt.org/directory"}},
						ExternalDns:         config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						ExternalSecrets:     config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						KubePrometheusStack: config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						IngressNginx:        config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						Kyverno:             config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						KyvernoPolicies:     config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						KyvernoPolicyReport: config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						Loki:                config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						HomerDashboard:      config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						Oauth2Proxy:         config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						MetricsServer:       config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						MetalLb:             config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
						Longhorn:            config.GenericService{ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled}},
					},
				})

				// Add global flags
				globalFlags := []string{
					"--config-file", configPath,
					"--work-dir", tempDir,
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

func createTestApp(commands ...*cli.Command) *cli.Command {
	return &cli.Command{
		Name:     "kubara",
		Commands: commands,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config-file",
				Usage: "Path to the configuration file",
			},
			&cli.StringFlag{
				Name:  "work-dir",
				Usage: "Working directory",
			},
		},
	}
}
