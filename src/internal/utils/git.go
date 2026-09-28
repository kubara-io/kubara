package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindGitRepoRoot traverses upwards from startDir looking for a .git directory or file
// (supporting standard repositories, worktrees, and git submodules).
// If no git repository root is found, it returns an empty string and an error.
func FindGitRepoRoot(startDir string) (string, error) {
	absDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path for %q: %w", startDir, err)
	}

	current := absDir
	for {
		gitPath := filepath.Join(current, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root
			break
		}
		current = parent
	}

	return "", fmt.Errorf("not inside of a git repository")
}

// ComputeGitRelativePath returns the forward-slash relative path from gitRoot to workspaceDir.
// If gitRoot is empty, workspaceDir is outside gitRoot, or workspaceDir equals gitRoot, it returns "".
func ComputeGitRelativePath(gitRoot, workspaceDir string) string {
	if gitRoot == "" || workspaceDir == "" {
		return ""
	}

	absGitRoot, err := filepath.Abs(gitRoot)
	if err != nil {
		return ""
	}
	absWorkspace, err := filepath.Abs(workspaceDir)
	if err != nil {
		return ""
	}

	rel, err := filepath.Rel(absGitRoot, absWorkspace)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return ""
	}

	return filepath.ToSlash(filepath.Clean(rel))
}

// ComputeGitOpsRepoPaths computes the git repository relative paths for components and configs.
// When gitRelPath is empty, standard root paths are returned.
func ComputeGitOpsRepoPaths(gitRelPath string) (componentsPath, configsPath string) {
	if gitRelPath == "" {
		return "platform-components/helm", "platform-configs"
	}
	componentsPath = filepath.ToSlash(filepath.Clean(filepath.Join(gitRelPath, "platform-components", "helm")))
	configsPath = filepath.ToSlash(filepath.Clean(filepath.Join(gitRelPath, "platform-configs")))
	return componentsPath, configsPath
}

// FindGitConfigPath returns the absolute path to .git/config for the repository containing startDir.
// If startDir is not in a git repository or the config file does not exist, it returns an empty string and no error.
func FindGitConfigPath(startDir string) (string, error) {
	if startDir == "" {
		return "", nil
	}
	gitRoot, err := FindGitRepoRoot(startDir)
	if err != nil || gitRoot == "" {
		return "", err
	}

	gitPath := filepath.Join(gitRoot, ".git")
	fi, err := os.Stat(gitPath)
	if err != nil {
		return "", nil
	}

	if fi.IsDir() {
		configPath := filepath.Join(gitPath, "config")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		return "", nil
	}

	// .git is a file (e.g. worktree or submodule)
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", err
	}

	line := strings.TrimSpace(string(data))
	if strings.HasPrefix(line, "gitdir:") {
		gitDir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(gitRoot, gitDir)
		}
		configPath := filepath.Join(gitDir, "config")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		commondirPath := filepath.Join(gitDir, "commondir")
		if cdata, err := os.ReadFile(commondirPath); err == nil {
			commonDir := strings.TrimSpace(string(cdata))
			if !filepath.IsAbs(commonDir) {
				commonDir = filepath.Join(gitDir, commonDir)
			}
			commonConfig := filepath.Join(commonDir, "config")
			if _, err := os.Stat(commonConfig); err == nil {
				return commonConfig, nil
			}
		}
	}

	return "", nil
}
