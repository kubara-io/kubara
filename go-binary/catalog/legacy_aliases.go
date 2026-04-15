package catalog

import "strings"

var legacyToCanonicalServiceName = map[string]string{
	// canonical keys
	"argo-cd":                 "argo-cd",
	"cert-manager":            "cert-manager",
	"external-dns":            "external-dns",
	"external-secrets":        "external-secrets",
	"kube-prometheus-stack":   "kube-prometheus-stack",
	"traefik":                 "traefik",
	"kyverno":                 "kyverno",
	"kyverno-policies":        "kyverno-policies",
	"kyverno-policy-reporter": "kyverno-policy-reporter",
	"loki":                    "loki",
	"homer-dashboard":         "homer-dashboard",
	"oauth2-proxy":            "oauth2-proxy",
	"metrics-server":          "metrics-server",
	"metallb":                 "metallb",
	"longhorn":                "longhorn",

	// legacy camelCase keys
	"argocd":              "argo-cd",
	"certmanager":         "cert-manager",
	"externaldns":         "external-dns",
	"externalsecrets":     "external-secrets",
	"kubeprometheusstack": "kube-prometheus-stack",
	"kyvernopolicies":     "kyverno-policies",
	"kyvernopolicyreport": "kyverno-policy-reporter",
	"homerdashboard":      "homer-dashboard",
	"oauth2proxy":         "oauth2-proxy",
	"metricsserver":       "metrics-server",
	"metalb":              "metallb",
	"metallb-old":         "metallb",
	"metallb_legacy":      "metallb",
	"metal-lb":            "metallb",
	"metalLb":             "metallb",
}

var canonicalToLegacyServiceName = map[string]string{
	"argo-cd":                 "argocd",
	"cert-manager":            "certManager",
	"external-dns":            "externalDns",
	"external-secrets":        "externalSecrets",
	"kube-prometheus-stack":   "kubePrometheusStack",
	"traefik":                 "traefik",
	"kyverno":                 "kyverno",
	"kyverno-policies":        "kyvernoPolicies",
	"kyverno-policy-reporter": "kyvernoPolicyReport",
	"loki":                    "loki",
	"homer-dashboard":         "homerDashboard",
	"oauth2-proxy":            "oauth2Proxy",
	"metrics-server":          "metricsServer",
	"metallb":                 "metalLb",
	"longhorn":                "longhorn",
}

func CanonicalServiceName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	if canonical, ok := legacyToCanonicalServiceName[trimmed]; ok {
		return canonical
	}
	if canonical, ok := legacyToCanonicalServiceName[strings.ToLower(trimmed)]; ok {
		return canonical
	}
	return trimmed
}

func LegacyServiceAliasMap() map[string]string {
	out := make(map[string]string, len(canonicalToLegacyServiceName))
	for canonical, legacy := range canonicalToLegacyServiceName {
		out[canonical] = legacy
	}
	return out
}
