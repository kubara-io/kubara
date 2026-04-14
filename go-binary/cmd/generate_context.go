package cmd

import "kubara/assets/config"

const (
	annotationCertManagerClusterIssuer = "cert-manager.io/cluster-issuer"
	annotationExternalDNSTarget        = "external-dns.alpha.kubernetes.io/target"
	annotationTraefikRouterMiddleware  = "traefik.ingress.kubernetes.io/router.middlewares"
	annotationTraefikAppRoot           = "traefik.ingress.kubernetes.io/app-root"

	traefikMiddlewareOauth2      = "oauth2-proxy-oauth-auth@kubernetescrd"
	traefikMiddlewareKyvernoPath = "oauth2-proxy-oauth-auth@kubernetescrd,\nkyverno-policy-reporter-strip-kyverno-prefix@kubernetescrd"
	traefikKyvernoAppRoot        = "/kyverno/#/"
)

func buildTemplateComputed(cluster config.Cluster) map[string]any {
	oauth2Enabled := cluster.Services.Oauth2Proxy.Status == config.StatusEnabled
	traefikEnabled := cluster.Services.Traefik.Status == config.StatusEnabled
	metalLbEnabled := cluster.Services.MetalLb.Status == config.StatusEnabled
	traefikAuthEnabled := oauth2Enabled && traefikEnabled

	argocdOverrides := ingressOverrides(cluster.Services.Argocd.Ingress)
	homerOverrides := ingressOverrides(cluster.Services.HomerDashboard.Ingress)
	kubePromOverrides := ingressOverrides(cluster.Services.KubePrometheusStack.Ingress)
	kyvernoPolicyReporterOverrides := ingressOverrides(cluster.Services.KyvernoPolicyReport.Ingress)
	longhornOverrides := ingressOverrides(cluster.Services.Longhorn.Ingress)
	oauth2ProxyOverrides := ingressOverrides(cluster.Services.Oauth2Proxy.Ingress)

	argocdIngressDefaults := standardIngressAnnotations(cluster, metalLbEnabled, traefikAuthEnabled)
	argocdGrpcDefaults := baseIngressAnnotations(cluster, metalLbEnabled)

	homerDefaults := standardIngressAnnotations(cluster, metalLbEnabled, traefikAuthEnabled)

	kubePromDefaults := standardIngressAnnotations(cluster, metalLbEnabled, traefikAuthEnabled)

	kyvernoPolicyReporterDefaults := baseIngressAnnotations(cluster, metalLbEnabled)
	if traefikAuthEnabled {
		kyvernoPolicyReporterDefaults[annotationTraefikRouterMiddleware] = traefikMiddlewareKyvernoPath
		kyvernoPolicyReporterDefaults[annotationTraefikAppRoot] = traefikKyvernoAppRoot
	}

	longhornDefaults := standardIngressAnnotations(cluster, metalLbEnabled, traefikAuthEnabled)

	oauth2ProxyDefaults := baseIngressAnnotations(cluster, metalLbEnabled)

	return map[string]any{
		"ingressAnnotations": map[string]any{
			"argocd": map[string]any{
				"ingress": mergeStringMaps(argocdIngressDefaults, argocdOverrides),
				"grpc":    mergeStringMaps(argocdGrpcDefaults, argocdOverrides),
			},
			"homerDashboard":      mergeStringMaps(homerDefaults, homerOverrides),
			"kubePrometheusStack": mergeStringMaps(kubePromDefaults, kubePromOverrides),
			"kyvernoPolicyReport": mergeStringMaps(kyvernoPolicyReporterDefaults, kyvernoPolicyReporterOverrides),
			"longhorn":            mergeStringMaps(longhornDefaults, longhornOverrides),
			"oauth2Proxy":         mergeStringMaps(oauth2ProxyDefaults, oauth2ProxyOverrides),
		},
	}
}

func baseIngressAnnotations(cluster config.Cluster, includeExternalDNSTarget bool) map[string]string {
	annotations := map[string]string{
		annotationCertManagerClusterIssuer: cluster.Services.CertManager.ClusterIssuer.Name,
	}
	if includeExternalDNSTarget {
		annotations[annotationExternalDNSTarget] = cluster.PublicLoadBalancerIP
	}
	return annotations
}

func standardIngressAnnotations(cluster config.Cluster, includeExternalDNSTarget bool, includeTraefikMiddleware bool) map[string]string {
	annotations := baseIngressAnnotations(cluster, includeExternalDNSTarget)
	if includeTraefikMiddleware {
		annotations[annotationTraefikRouterMiddleware] = traefikMiddlewareOauth2
	}
	return annotations
}

func ingressOverrides(overrides *config.IngressOverrides) map[string]string {
	if overrides == nil || len(overrides.Annotations) == 0 {
		return map[string]string{}
	}
	merged := make(map[string]string, len(overrides.Annotations))
	for key, value := range overrides.Annotations {
		merged[key] = value
	}
	return merged
}

func mergeStringMaps(defaults map[string]string, overrides map[string]string) map[string]string {
	merged := make(map[string]string, len(defaults)+len(overrides))
	for key, value := range defaults {
		merged[key] = value
	}
	for key, value := range overrides {
		merged[key] = value
	}
	return merged
}
