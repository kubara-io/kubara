package utils

import "strings"

// NormalizeHelmRepoURL normalizes Helm repository inputs for ArgoCD.
// If oci:// is provided, it is removed because ArgoCD helm repository
// credentials expect the registry URL without the scheme.
func NormalizeHelmRepoURL(v string) string {
	trimmed := strings.TrimSpace(v)
	if strings.HasPrefix(strings.ToLower(trimmed), "oci://") {
		return trimmed[len("oci://"):]
	}
	return trimmed
}

// IsOCIHelmRepoURL reports whether a Helm repository URL should be treated
// as OCI. HTTPS/HTTP URLs are treated as classic Helm repos.
func IsOCIHelmRepoURL(v string) bool {
	normalized := NormalizeHelmRepoURL(v)
	if normalized == "" {
		return false
	}
	lower := strings.ToLower(normalized)
	return !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "http://")
}
