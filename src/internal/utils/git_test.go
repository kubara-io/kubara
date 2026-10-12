package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGitRepoRoot(t *testing.T) {
	// 1. Directory with .git directory
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	subDir := filepath.Join(tempDir, "setups", "dev-fleet")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create setups dir: %v", err)
	}

	root, err := FindGitRepoRoot(subDir)
	if err != nil {
		t.Fatalf("FindGitRepoRoot failed: %v", err)
	}
	if root != tempDir {
		t.Errorf("expected root %q, got %q", tempDir, root)
	}

	// 2. Directory with .git file (submodule / worktree)
	tempDir2 := t.TempDir()
	gitFile := filepath.Join(tempDir2, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: /some/path\n"), 0644); err != nil {
		t.Fatalf("failed to write .git file: %v", err)
	}

	nestedDir := filepath.Join(tempDir2, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	root2, err := FindGitRepoRoot(nestedDir)
	if err != nil {
		t.Fatalf("FindGitRepoRoot failed: %v", err)
	}
	if root2 != tempDir2 {
		t.Errorf("expected root %q, got %q", tempDir2, root2)
	}

	// 3. No .git found (tempdir without .git)
	tempDir3 := t.TempDir()
	nonGitDir := filepath.Join(tempDir3, "no-git", "sub")
	if err := os.MkdirAll(nonGitDir, 0755); err != nil {
		t.Fatalf("failed to create non-git dir: %v", err)
	}

	root3, err := FindGitRepoRoot(nonGitDir)
	if err == nil {
		t.Fatalf("expected error from FindGitRepoRoot on non-git dir, got nil")
	}
	if root3 != "" {
		t.Errorf("expected empty root, got %q", root3)
	}
}

func TestComputeGitRelativePath(t *testing.T) {
	tests := []struct {
		name         string
		gitRoot      string
		workspaceDir string
		expected     string
	}{
		{
			name:         "Empty git root",
			gitRoot:      "",
			workspaceDir: "/repo/setups/dev",
			expected:     "",
		},
		{
			name:         "Empty workspace",
			gitRoot:      "/repo",
			workspaceDir: "",
			expected:     "",
		},
		{
			name:         "Same directory as git root",
			gitRoot:      "/repo",
			workspaceDir: "/repo",
			expected:     "",
		},
		{
			name:         "Subdirectory",
			gitRoot:      "/repo",
			workspaceDir: "/repo/setups/dev-fleet",
			expected:     "setups/dev-fleet",
		},
		{
			name:         "Deep subdirectory",
			gitRoot:      "/repo",
			workspaceDir: "/repo/environments/eu/prod",
			expected:     "environments/eu/prod",
		},
		{
			name:         "Outside git root",
			gitRoot:      "/repo/dir",
			workspaceDir: "/other/dir",
			expected:     "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rel := ComputeGitRelativePath(tc.gitRoot, tc.workspaceDir)
			if rel != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, rel)
			}
		})
	}
}

func TestComputeGitOpsRepoPaths(t *testing.T) {
	tests := []struct {
		gitRelPath     string
		wantComponents string
		wantConfigs    string
	}{
		{
			gitRelPath:     "",
			wantComponents: "platform-components/helm",
			wantConfigs:    "platform-configs",
		},
		{
			gitRelPath:     "setups/dev-fleet",
			wantComponents: "setups/dev-fleet/platform-components/helm",
			wantConfigs:    "setups/dev-fleet/platform-configs",
		},
	}

	for _, tc := range tests {
		comp, conf := ComputeGitOpsRepoPaths(tc.gitRelPath)
		if comp != tc.wantComponents {
			t.Errorf("expected components %q, got %q", tc.wantComponents, comp)
		}
		if conf != tc.wantConfigs {
			t.Errorf("expected configs %q, got %q", tc.wantConfigs, conf)
		}
	}
}
