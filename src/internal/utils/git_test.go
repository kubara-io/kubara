package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSSHGitURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"scp syntax", "git@github.com:kubara-io/kubara.git", true},
		{"ssh protocol", "ssh://git@github.com/kubara-io/kubara.git", true},
		{"https protocol", "https://github.com/kubara-io/kubara.git", false},
		{"http protocol", "http://github.com/kubara-io/kubara.git", false},
		{"empty string", "", false},
		{"whitespace", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsSSHGitURL(tt.url))
		})
	}
}

func TestIsHTTPGitURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"https protocol", "https://github.com/kubara-io/kubara.git", true},
		{"http protocol", "http://github.com/kubara-io/kubara.git", true},
		{"scp syntax", "git@github.com:kubara-io/kubara.git", false},
		{"ssh protocol", "ssh://git@github.com/kubara-io/kubara.git", false},
		{"empty string", "", false},
		{"whitespace", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsHTTPGitURL(tt.url))
		})
	}
}
