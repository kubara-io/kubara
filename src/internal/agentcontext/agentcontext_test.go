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

func TestRenderPinsVersion(t *testing.T) {
	rendered, err := Render("v0.10.0")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	content := string(rendered)

	if !strings.Contains(content, "raw.githubusercontent.com/kubara-io/kubara/v0.10.0/docs/content/") {
		t.Errorf("AGENTS.md does not pin raw links to the version tag:\n%s", content)
	}
	if !strings.Contains(content, "v0.10.0") {
		t.Errorf("AGENTS.md does not mention the installed version")
	}
}

func TestRenderDevFallsBackToMain(t *testing.T) {
	rendered, err := Render("dev")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(string(rendered), "raw.githubusercontent.com/kubara-io/kubara/main/docs/content/") {
		t.Errorf("dev build should pin raw links to main:\n%s", rendered)
	}
	// The "pinned to" heading must reflect the resolved ref, not the display version.
	if strings.Contains(string(rendered), "pinned to `dev`") {
		t.Errorf("doc heading should pin to the resolved ref (main), not the display version 'dev'")
	}
	if !strings.Contains(string(rendered), "pinned to `main`") {
		t.Errorf("doc heading should state it is pinned to main for dev builds")
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

	docPathRe := regexp.MustCompile(`docs/content/[^)\s]+\.md`)
	seen := map[string]bool{}
	for _, match := range docPathRe.FindAllString(string(rendered), -1) {
		seen[match] = true
	}

	if len(seen) == 0 {
		t.Fatal("no docs/content links found in rendered template")
	}

	for docPath := range seen {
		full := filepath.Join(repoRoot, filepath.FromSlash(docPath))
		if _, err := os.Stat(full); err != nil {
			t.Errorf("referenced doc does not exist: %s (%v)", docPath, err)
		}
	}
}
