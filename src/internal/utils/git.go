package utils

import "strings"

// IsSSHGitURL reports whether a URL uses SSH syntax.
func IsSSHGitURL(value string) bool {
	trimmed := strings.TrimSpace(value)
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "ssh://") || (!strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") && strings.Contains(trimmed, "@"))
}

// IsHTTPGitURL reports whether a URL uses HTTP or HTTPS.
func IsHTTPGitURL(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://")
}
