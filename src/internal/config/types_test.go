package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTerraformProviderIsSupported(t *testing.T) {
	for _, provider := range SupportedTerraformProviders() {
		assert.True(t, provider.IsSupported(), "listed provider %q must be supported", provider)
	}

	assert.False(t, TerraformProviderPlaceholder.IsSupported())
	assert.False(t, TerraformProvider("unknown").IsSupported())
}
