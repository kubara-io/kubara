package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/service"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
	"sigs.k8s.io/yaml"
)

func CreateTestServices() service.Services {
	return service.Services{
		"argocd":                  {Status: service.StatusEnabled},
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

func CreateTestConfig(t *testing.T, dir string, clusters ...config.Cluster) string {
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

func CreateTestCluster(t *testing.T) config.Cluster {
	t.Helper()
	return config.Cluster{
		Name:    "missing-terraform-cluster",
		Stage:   "dev",
		Type:    "hub",
		DNSName: "test.example.com",
		ArgoCD: config.ArgoCD{
			Repo: config.RepoProto{
				HTTPS: &config.RepoType{
					Configs:    config.Repository{URL: "https://github.com/example/configs", TargetRevision: "main"},
					Components: config.Repository{URL: "https://github.com/example/components", TargetRevision: "main"},
				},
			},
		},
		Services: CreateTestServices(),
	}
}

func CreateDefaultGenerateTestEnv(t *testing.T, dir string) string {
	t.Helper()

	return CreateTestEnv(t, dir, envconfig.EnvMap{
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
	})
}

// createTestEnv writes an envMap to the file system
// It returns the file path
// Takes a directory and an EnvMap and validates the envMap before writing it
func CreateTestEnv(t *testing.T, dir string, env envconfig.EnvMap) string {
	envPath := filepath.Join(dir, ".env")
	err := env.Validate()
	require.NoError(t, err)

	var b strings.Builder
	v := reflect.ValueOf(&env).Elem()
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := typ.Field(i)
		koanfKey := fieldType.Tag.Get("koanf")
		if koanfKey == "" {
			continue
		}
		fmt.Fprintf(&b, "%s='%v'\n", koanfKey, fieldVal.Interface())
	}

	err = os.WriteFile(envPath, []byte(b.String()), 0600)

	require.NoError(t, err)

	return envPath
}

func CreateTestAppWithFlags(flags []cli.Flag, commands ...*cli.Command) *cli.Command {
	return &cli.Command{
		Name:     "kubara",
		Commands: commands,
		Flags:    flags,
	}
}
