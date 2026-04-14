package cmd

import (
	"testing"

	"kubara/assets/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTemplateComputed_MergesDefaultsAndOverrides(t *testing.T) {
	cluster := config.Cluster{
		PublicLoadBalancerIP: "1.2.3.4",
		Services: config.Services{
			CertManager: config.CertManagerService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
				ClusterIssuer: config.ClusterIssuer{Name: "letsencrypt-prod"},
			},
			Oauth2Proxy: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			Traefik: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			MetalLb: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			HomerDashboard: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
				Ingress: &config.IngressOverrides{
					Annotations: map[string]string{
						annotationCertManagerClusterIssuer: "letsencrypt-custom",
					},
				},
			},
			Argocd: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
				Ingress: &config.IngressOverrides{
					Annotations: map[string]string{
						"nginx.ingress.kubernetes.io/rewrite-target": "/",
					},
				},
			},
			KubePrometheusStack: config.PersistentService{
				GenericService: config.GenericService{
					ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
				},
			},
			KyvernoPolicyReport: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			Longhorn: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
		},
	}

	computed := buildTemplateComputed(cluster)
	ingressAnnotations, ok := computed["ingressAnnotations"].(map[string]any)
	require.True(t, ok)

	homer, ok := ingressAnnotations["homerDashboard"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "letsencrypt-custom", homer[annotationCertManagerClusterIssuer])
	assert.Equal(t, "1.2.3.4", homer[annotationExternalDNSTarget])
	assert.Equal(t, traefikMiddlewareOauth2, homer[annotationTraefikRouterMiddleware])

	argocd, ok := ingressAnnotations["argocd"].(map[string]any)
	require.True(t, ok)
	argocdIngress, ok := argocd["ingress"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, traefikMiddlewareOauth2, argocdIngress[annotationTraefikRouterMiddleware])
	assert.Equal(t, "/", argocdIngress["nginx.ingress.kubernetes.io/rewrite-target"])

	argocdGrpc, ok := argocd["grpc"].(map[string]string)
	require.True(t, ok)
	_, hasMiddleware := argocdGrpc[annotationTraefikRouterMiddleware]
	assert.False(t, hasMiddleware)
}

func TestBuildTemplateComputed_SkipsTraefikAnnotationsWhenTraefikDisabled(t *testing.T) {
	cluster := config.Cluster{
		Services: config.Services{
			CertManager: config.CertManagerService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
				ClusterIssuer: config.ClusterIssuer{Name: "letsencrypt-prod"},
			},
			Oauth2Proxy: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			Traefik: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusDisabled},
			},
			HomerDashboard: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			Argocd: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			KubePrometheusStack: config.PersistentService{
				GenericService: config.GenericService{
					ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
				},
			},
			KyvernoPolicyReport: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			Longhorn: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
		},
	}

	computed := buildTemplateComputed(cluster)
	ingressAnnotations, ok := computed["ingressAnnotations"].(map[string]any)
	require.True(t, ok)

	homer, ok := ingressAnnotations["homerDashboard"].(map[string]string)
	require.True(t, ok)
	_, hasHomerMiddleware := homer[annotationTraefikRouterMiddleware]
	assert.False(t, hasHomerMiddleware)

	argocd, ok := ingressAnnotations["argocd"].(map[string]any)
	require.True(t, ok)
	argocdIngress, ok := argocd["ingress"].(map[string]string)
	require.True(t, ok)
	_, hasArgocdMiddleware := argocdIngress[annotationTraefikRouterMiddleware]
	assert.False(t, hasArgocdMiddleware)
}

func TestBuildTemplateComputed_SkipsExternalDNSTargetWhenMetalLbDisabled(t *testing.T) {
	cluster := config.Cluster{
		PublicLoadBalancerIP: "1.2.3.4",
		Services: config.Services{
			CertManager: config.CertManagerService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
				ClusterIssuer: config.ClusterIssuer{Name: "letsencrypt-prod"},
			},
			Oauth2Proxy: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			Traefik: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			MetalLb: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusDisabled},
			},
			HomerDashboard: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			Argocd: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			KubePrometheusStack: config.PersistentService{
				GenericService: config.GenericService{
					ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
				},
			},
			KyvernoPolicyReport: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
			Longhorn: config.GenericService{
				ServiceStatus: config.ServiceStatus{Status: config.StatusEnabled},
			},
		},
	}

	computed := buildTemplateComputed(cluster)
	ingressAnnotations, ok := computed["ingressAnnotations"].(map[string]any)
	require.True(t, ok)

	homer, ok := ingressAnnotations["homerDashboard"].(map[string]string)
	require.True(t, ok)
	_, hasHomerExternalDNS := homer[annotationExternalDNSTarget]
	assert.False(t, hasHomerExternalDNS)

	argocd, ok := ingressAnnotations["argocd"].(map[string]any)
	require.True(t, ok)
	argocdIngress, ok := argocd["ingress"].(map[string]string)
	require.True(t, ok)
	_, hasArgocdExternalDNS := argocdIngress[annotationExternalDNSTarget]
	assert.False(t, hasArgocdExternalDNS)

	argocdGrpc, ok := argocd["grpc"].(map[string]string)
	require.True(t, ok)
	_, hasArgocdGrpcExternalDNS := argocdGrpc[annotationExternalDNSTarget]
	assert.False(t, hasArgocdGrpcExternalDNS)
}
