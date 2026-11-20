package generate

import (
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

func TestBuildTemplateContext_IncludesEmptyHelmRepoWhenUnconfigured(t *testing.T) {
	context, err := buildTemplateContext(config.Cluster{
		ArgoCD: &config.ArgoCD{
			Repo: config.RepoProto{
				Git: &config.RepoType{},
			},
		},
	}, buildContext{})

	if !assert.NoError(t, err) {
		return
	}

	cluster := context["cluster"].(map[string]any)
	argocd := cluster["argocd"].(map[string]any)
	assert.Equal(t, map[string]any{"url": ""}, argocd["helmRepo"])
}

func TestBuildTemplateContext_IncludesGitOpsNamespace(t *testing.T) {
	context, err := buildTemplateContext(config.Cluster{
		Name: "test-hub",
		Type: config.Hub,
		ArgoCD: &config.ArgoCD{
			Repo: config.RepoProto{
				Git: &config.RepoType{
					Configs: config.Repository{
						URL: "https://git.example.com/configs.git",
					},
				},
			},
		},
	}, buildContext{
		GitOps: &config.GitOps{
			Engine:      config.GitOpsEngineArgoCD,
			SelfManaged: config.ArgoCDSelfManagedEnabled,
			Repo: config.RepoProto{
				Git: &config.RepoType{
					Configs: config.Repository{
						URL: "https://git.example.com/configs.git",
					},
				},
			},
		},
	})

	require.NoError(t, err)
	gitops, ok := context["gitops"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "argocd", gitops["engine"])
	assert.Equal(t, "enabled", gitops["selfManaged"])
}

func TestBuildTemplateContext_SynthesizesClusterArgoCDFromGitOps(t *testing.T) {
	context, err := buildTemplateContext(config.Cluster{
		Name: "test-hub",
		Type: config.Hub,
	}, buildContext{
		GitOps: &config.GitOps{
			Engine:      config.GitOpsEngineArgoCD,
			SelfManaged: config.ArgoCDSelfManagedEnabled,
			Repo: config.RepoProto{
				Git: &config.RepoType{
					Configs: config.Repository{
						URL: "https://git.example.com/configs.git",
					},
				},
			},
		},
	})

	require.NoError(t, err)
	cluster, ok := context["cluster"].(map[string]any)
	require.True(t, ok)
	argocd, ok := cluster["argocd"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "enabled", argocd["selfManaged"])
	repo, ok := argocd["repo"].(map[string]any)
	require.True(t, ok)
	git, ok := repo["git"].(map[string]any)
	require.True(t, ok)
	configs, ok := git["configs"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://git.example.com/configs.git", configs["url"])
}
