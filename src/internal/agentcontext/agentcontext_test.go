package agentcontext

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDocsRef(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{name: "empty falls back to main", version: "", want: "main"},
		{name: "dev falls back to main", version: "dev", want: "main"},
		{name: "release tag with v prefix", version: "v0.10.0", want: "v0.10.0"},
		{name: "release tag without v prefix", version: "0.10.0", want: "v0.10.0"},
		{name: "whitespace is trimmed", version: "  v1.2.3  ", want: "v1.2.3"},
		{name: "pre-release falls back to main", version: "v1.2.3-rc1", want: "main"},
		{name: "snapshot pseudo version falls back to main", version: "v0.10.1-next+abc123", want: "main"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DocsRef(tt.version); got != tt.want {
				t.Errorf("DocsRef(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

func TestDocsVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{name: "release tag with v prefix", version: "v0.11.0", want: "v0.11.0"},
		{name: "release tag without v prefix", version: "0.11.0", want: "v0.11.0"},
		{name: "empty maps to latest-dev", version: "", want: "latest-dev"},
		{name: "dev maps to latest-dev", version: "dev", want: "latest-dev"},
		{name: "pre-release maps to latest-dev", version: "v1.2.3-rc1", want: "latest-dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DocsVersion(tt.version); got != tt.want {
				t.Errorf("DocsVersion(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

func TestRenderPinsVersion(t *testing.T) {
	rendered, err := Render("v0.10.0")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	content := string(rendered)

	// Curated links point at this version's published Markdown on the docs site.
	if !strings.Contains(content, "https://docs.kubara.io/v0.10.0/1_getting_started/prerequisites/index.md") {
		t.Errorf("AGENTS.md does not link curated docs at the version's hosted path:\n%s", content)
	}
	// The docs index pointer must target this version's hosted llms.txt.
	if !strings.Contains(content, "https://docs.kubara.io/v0.10.0/llms.txt") {
		t.Errorf("AGENTS.md does not link the version's llms.txt index:\n%s", content)
	}
	if !strings.Contains(content, "v0.10.0") {
		t.Errorf("AGENTS.md does not mention the installed version")
	}
}

func TestRenderDevUsesLatestDevDocs(t *testing.T) {
	rendered, err := Render("dev")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	content := string(rendered)

	// Dev/snapshot builds have no published version; docs links use the
	// latest-dev alias ("main" is a git ref, not a published docs version).
	if !strings.Contains(content, "https://docs.kubara.io/latest-dev/1_getting_started/prerequisites/index.md") {
		t.Errorf("dev build should link curated docs at latest-dev:\n%s", content)
	}
	if !strings.Contains(content, "https://docs.kubara.io/latest-dev/llms.txt") {
		t.Errorf("dev build should link the latest-dev llms.txt index:\n%s", content)
	}
	// Docs are served from the docs site, not rate-limited raw GitHub.
	if strings.Contains(content, "raw.githubusercontent.com") {
		t.Errorf("AGENTS.md should not link raw GitHub:\n%s", content)
	}
}

func TestWriteSkipsExistingUnlessOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, AgentsFileName)

	result, err := Write(dir, "v0.10.0", false)
	if err != nil {
		t.Fatalf("first Write returned error: %v", err)
	}
	if !result.Written {
		t.Errorf("expected file to be written on first run")
	}
	if result.Path != path {
		t.Errorf("unexpected path: got %q, want %q", result.Path, path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}

	// Mutate the file, then confirm a non-overwrite run leaves it untouched.
	sentinel := []byte("user edit, keep me")
	if err := os.WriteFile(path, sentinel, 0o644); err != nil {
		t.Fatalf("failed to mutate file: %v", err)
	}

	result, err = Write(dir, "v0.10.0", false)
	if err != nil {
		t.Fatalf("second Write returned error: %v", err)
	}
	if result.Written {
		t.Errorf("expected file to be skipped on non-overwrite run")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after skip: %v", err)
	}
	if string(got) != string(sentinel) {
		t.Errorf("non-overwrite run modified an existing file")
	}

	// Overwrite refreshes the file.
	result, err = Write(dir, "v0.10.0", true)
	if err != nil {
		t.Fatalf("overwrite Write returned error: %v", err)
	}
	if !result.Written {
		t.Errorf("expected file to be written on overwrite run")
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after overwrite: %v", err)
	}
	if string(got) == string(sentinel) {
		t.Errorf("overwrite run did not refresh the file")
	}
}

// TestRenderedDocLinksExist guards against documentation drift: every docs/content
// path referenced from the embedded template must exist in the repository, so a
// docs reorganization that breaks these links fails the test suite.
func TestRenderedDocLinksExist(t *testing.T) {
	// repoRoot is three levels up from this package (src/internal/agentcontext).
	// The docs tree is not checked out in src-only (sparse) CI jobs, so skip there;
	// the docs-check workflow runs this guard with the full tree present.
	repoRoot := filepath.Join("..", "..", "..")
	if _, err := os.Stat(filepath.Join(repoRoot, "docs", "content")); err != nil {
		t.Skip("docs tree not present (e.g. sparse checkout); link guard runs in the docs-check workflow")
	}

	rendered, err := Render("main")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	// Curated links point at published Markdown:
	//   https://docs.kubara.io/<version>/<page>/index.md
	// published from the source docs/content/<page>.md.
	docURLRe := regexp.MustCompile(`docs\.kubara\.io/[^/]+/([^)\s]+)/index\.md`)
	seen := map[string]bool{}
	for _, m := range docURLRe.FindAllStringSubmatch(string(rendered), -1) {
		seen[m[1]] = true
	}

	if len(seen) == 0 {
		t.Fatal("no docs links found in rendered template")
	}

	for page := range seen {
		full := filepath.Join(repoRoot, "docs", "content", filepath.FromSlash(page)+".md")
		if _, err := os.Stat(full); err != nil {
			t.Errorf("referenced doc does not exist: docs/content/%s.md (%v)", page, err)
		}
	}
}
