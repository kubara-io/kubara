package generate

import (
	"path/filepath"
	"testing"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/render"
	"github.com/kubara-io/kubara/internal/service"

	"github.com/stretchr/testify/assert"
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

func TestBuildEnabledServiceTemplatePathPredicate_SkipsHubOnlyBootstrapServiceForSpoke(t *testing.T) {
	predicate := buildServiceTemplateFilter(
		config.Cluster{
			Type: config.Spoke,
		},
		catalog.Catalog{
			Services: map[string]catalog.ServiceDefinition{
				"argo-cd": {
					Spec: catalog.ServiceSpec{
						ChartPath:    "argo-cd",
						ClusterTypes: []string{"hub"},
					},
				},
				"cluster-credential-rotator": {
					Spec: catalog.ServiceSpec{
						ChartPath:    "cluster-credential-rotator",
						ClusterTypes: []string{"hub"},
					},
				},
			},
		},
	)

	assert.False(t, predicate(filepath.Join(render.DefaultPlatformConfigsPath, render.Helm.String(), "argo-cd", "values.generated.yaml.tplt")))
	assert.False(t, predicate(filepath.Join(render.DefaultPlatformConfigsPath, render.Helm.String(), "cluster-credential-rotator", "values.generated.yaml.tplt")))
}
