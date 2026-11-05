package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeHelmRepoURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"oci prefix stripped", "oci://registry-1.docker.io/bitnamicharts", "registry-1.docker.io/bitnamicharts"},
		{"OCI uppercase stripped", "OCI://registry-1.docker.io/bitnamicharts", "registry-1.docker.io/bitnamicharts"},
		{"https unchanged", "https://charts.bitnami.com/bitnami", "https://charts.bitnami.com/bitnami"},
		{"bare domain unchanged", "registry-1.docker.io/bitnamicharts", "registry-1.docker.io/bitnamicharts"},
		{"empty unchanged", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, NormalizeHelmRepoURL(tt.value))
		})
	}
}

func TestIsOCIHelmRepoURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"oci prefix", "oci://registry-1.docker.io/bitnamicharts", true},
		{"bare domain", "registry-1.docker.io/bitnamicharts", true},
		{"https repo", "https://charts.bitnami.com/bitnami", false},
		{"http repo", "http://charts.bitnami.com/bitnami", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsOCIHelmRepoURL(tt.value))
		})
	}
}
