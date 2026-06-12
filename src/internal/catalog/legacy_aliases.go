package catalog

import "strings"

var legacyToCanonicalServiceName = map[string]string{
	// canonical keys
	"argocd":                  "argocd",
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
