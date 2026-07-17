package render

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateFiles_PathPredicateUsesProviderStrippedPath(t *testing.T) {
	results, err := TemplateFiles(TemplateOptions{
		Type:     Terraform,
		Provider: "stackit",
		Catalogs: testTemplateCatalogs(),
		Data: map[string]any{
			"var": map[string]any{
				"project_id": "12345",
				"name":       "tf-cluster",
				"stage":      "staging",
			},
			"cluster": map[string]any{
				"services": fullServiceContext(),
				"terraform": map[string]any{
					"kubernetesType": "ske",
				},
			},
			"catalog": fullCatalogContext(),
		},
		PathPredicate: func(path string) bool {
			return strings.HasPrefix(path, "platform-configs/terraform/")
		},
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, results)
}
