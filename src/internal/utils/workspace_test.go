package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWorkspace_Default(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git: %v", err)
	}

	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("version: v1alpha4\n"), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	ws, err := ResolveWorkspace(WorkspaceOptions{
		WorkDir: tempDir,
	})
	if err != nil {
		t.Fatalf("ResolveWorkspace failed: %v", err)
	}

	if ws.WorkDir != tempDir {
		t.Errorf("expected WorkDir to be %q, got %q", tempDir, ws.WorkDir)
	}
	if ws.ConfigFilePath != configFile {
		t.Errorf("expected ConfigFilePath to be %q, got %q", configFile, ws.ConfigFilePath)
	}
	if ws.GitRoot != tempDir {
		t.Errorf("expected GitRoot to be %q, got %q", tempDir, ws.GitRoot)
	}
	if ws.GitRelPath != "" {
		t.Errorf("expected GitRelPath to be empty, got %q", ws.GitRelPath)
	}
}

func TestResolveWorkspace_HubFlag(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git: %v", err)
	}

	setupDir := filepath.Join(tempDir, "prod")
	if err := os.MkdirAll(setupDir, 0755); err != nil {
		t.Fatalf("failed to create prod: %v", err)
	}

	configFile := filepath.Join(setupDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("version: v1alpha4\n"), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// Pass Hub="prod" from tempDir (repo root)
	ws, err := ResolveWorkspace(WorkspaceOptions{
		WorkDir: tempDir,
		Hub:     "prod",
		HubSet:  true,
	})
	if err != nil {
		t.Fatalf("ResolveWorkspace failed: %v", err)
	}

	if ws.WorkDir != setupDir {
		t.Errorf("expected WorkDir to be %q, got %q", setupDir, ws.WorkDir)
	}
	if ws.ConfigFilePath != configFile {
		t.Errorf("expected ConfigFilePath to be %q, got %q", configFile, ws.ConfigFilePath)
	}
	if ws.GitRoot != tempDir {
		t.Errorf("expected GitRoot to be %q, got %q", tempDir, ws.GitRoot)
	}
	if ws.GitRelPath != "prod" {
		t.Errorf("expected GitRelPath to be %q, got %q", "prod", ws.GitRelPath)
	}
}

func TestResolveWorkspace_EnvFallback(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git: %v", err)
	}

	rootEnv := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(rootEnv, []byte("KUBARA_ROOT=1\n"), 0644); err != nil {
		t.Fatalf("failed to write root .env: %v", err)
	}

	setupDir := filepath.Join(tempDir, "setups", "staging")
	if err := os.MkdirAll(setupDir, 0755); err != nil {
		t.Fatalf("failed to create setups/staging: %v", err)
	}

	// 1. When workspace doesn't have .env, fall back to root .env
	ws, err := ResolveWorkspace(WorkspaceOptions{
		WorkDir: setupDir,
	})
	if err != nil {
		t.Fatalf("ResolveWorkspace failed: %v", err)
	}
	if ws.EnvFilePath != rootEnv {
		t.Errorf("expected EnvFilePath to fall back to root %q, got %q", rootEnv, ws.EnvFilePath)
	}

	// 2. When workspace HAS .env, prefer workspace .env
	wsEnv := filepath.Join(setupDir, ".env")
	if err := os.WriteFile(wsEnv, []byte("KUBARA_STAGING=1\n"), 0644); err != nil {
		t.Fatalf("failed to write workspace .env: %v", err)
	}

	ws2, err := ResolveWorkspace(WorkspaceOptions{
		WorkDir: setupDir,
	})
	if err != nil {
		t.Fatalf("ResolveWorkspace failed: %v", err)
	}
	if ws2.EnvFilePath != wsEnv {
		t.Errorf("expected EnvFilePath to prefer workspace %q, got %q", wsEnv, ws2.EnvFilePath)
	}
}

func TestResolveWorkspace_HubPathValidation(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	require.NoError(t, os.Mkdir(gitDir, 0755))

	validSubdir := filepath.Join(tempDir, "setups", "dev")
	require.NoError(t, os.MkdirAll(validSubdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(validSubdir, "config.yaml"), []byte("version: v1alpha4\n"), 0644))

	validBetaSubdir := filepath.Join(tempDir, "setups", "v1.0..beta")
	require.NoError(t, os.MkdirAll(validBetaSubdir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(validBetaSubdir, "config.yaml"), []byte("version: v1alpha4\n"), 0644))

	outsideDir := t.TempDir()

	tests := []struct {
		name        string
		hub         string
		expectedDir string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid relative subpath",
			hub:         "setups/dev",
			expectedDir: validSubdir,
			wantErr:     false,
		},
		{
			name:        "valid subpath containing double dots in directory name",
			hub:         "setups/v1.0..beta",
			expectedDir: validBetaSubdir,
			wantErr:     false,
		},
		{
			name:        "empty hub path returns error",
			hub:         "  ",
			wantErr:     true,
			errContains: "hub path cannot be empty",
		},
		{
			name:        "parent directory traversal with .. prefix",
			hub:         "../outside",
			wantErr:     true,
			errContains: "cannot contain '..'",
		},
		{
			name:        "internal directory traversal containing ..",
			hub:         "setups/../dev",
			wantErr:     true,
			errContains: "cannot contain '..'",
		},
		{
			name:        "just ..",
			hub:         "..",
			wantErr:     true,
			errContains: "cannot contain '..'",
		},
		{
			name:        "absolute path outside CWD",
			hub:         outsideDir,
			wantErr:     true,
			errContains: "resolves outside current working directory",
		},
		{
			name:        "current directory dot",
			hub:         ".",
			wantErr:     true,
			errContains: "must resolve to a directory below current working directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, err := ResolveWorkspace(WorkspaceOptions{
				WorkDir: tempDir,
				Hub:     tt.hub,
				HubSet:  true,
			})
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedDir, ws.WorkDir)
		})
	}
}
