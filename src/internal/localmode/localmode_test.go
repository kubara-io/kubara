package localmode

import (
	"testing"

	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestApplyClusterProfileDisablesOAuth2ProxyForLocalMode(t *testing.T) {
	cluster := &config.Cluster{
		Services: service.Services{
			"cert-manager": {Status: service.StatusEnabled},
			"oauth2-proxy": {Status: service.StatusEnabled},
			"traefik":      {Status: service.StatusDisabled},
		},
	}
	ApplyClusterProfile(cluster, "local.example.test")
	ApplyClusterProfile(cluster, "local.example.test")

	assert.Equal(t, service.StatusEnabled, cluster.Services["cert-manager"].Status)
	assert.Equal(t, service.StatusDisabled, cluster.Services["oauth2-proxy"].Status)
	assert.Equal(t, service.StatusEnabled, cluster.Services["traefik"].Status)
	assert.Equal(t, "local.example.test", cluster.DNSName)
}
