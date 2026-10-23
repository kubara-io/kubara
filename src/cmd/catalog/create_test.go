package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v3"
)

func TestCatalogCreateCITemplates(t *testing.T) {
	for _, tt := range []struct {
		name  string
		flags []string
		paths []string
	}{
		{name: "default"},
		{name: "github", flags: []string{"--github-actions"}, paths: []string{".github/workflows/catalog.yaml"}},
		{name: "gitlab", flags: []string{"--gitlab-ci"}, paths: []string{".gitlab-ci.yml"}},
		{name: "both", flags: []string{"--github-actions", "--gitlab-ci"}, paths: []string{".github/workflows/catalog.yaml", ".gitlab-ci.yml"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			cmd := &cli.Command{Commands: []*cli.Command{NewCatalogCreate()}}
			args := append([]string{"kubara", "create"}, tt.flags...)
			if err := cmd.Run(context.Background(), append(args, "example")); err != nil {
				t.Fatal(err)
			}
			for _, path := range append([]string{"Catalog.yaml", "services", "platform-configs/helm", "platform-components/terraform"}, tt.paths...) {
				if _, err := os.Stat(filepath.Join("example", path)); err != nil {
					t.Errorf("missing scaffold path %s: %v", path, err)
				}
			}
			for _, path := range []string{".github/workflows/catalog.yaml", ".gitlab-ci.yml"} {
				wanted := false
				for _, expected := range tt.paths {
					wanted = wanted || path == expected
				}
				_, err := os.Stat(filepath.Join("example", path))
				if !wanted && !os.IsNotExist(err) {
					t.Errorf("unexpected CI file %s", path)
				}
			}
		})
	}
}

func TestCatalogCreatePreservesExistingDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir("example", 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("example", ".gitlab-ci.yml")
	if err := os.WriteFile(path, []byte("existing pipeline"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := createCatalog("example", true, true); err == nil {
		t.Fatal("expected existing directory error")
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "existing pipeline" {
		t.Fatalf("existing pipeline changed: %s, %v", contents, err)
	}
}

func TestCatalogCreateRejectsInvalidName(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, name := range []string{"../escape", "Uppercase", "-invalid", ""} {
		if err := createCatalog(name, true, true); err == nil {
			t.Errorf("expected invalid name error for %q", name)
		}
	}
}
