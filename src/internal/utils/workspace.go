package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"
)

// WorkspacePaths represents the resolved filesystem and git paths for a Kubara workspace.
type WorkspacePaths struct {
	WorkDir        string
	ConfigFilePath string
	EnvFilePath    string
	GitRoot        string
	GitRelPath     string
}

// WorkspaceOptions contains inputs for workspace resolution.
type WorkspaceOptions struct {
	WorkDir string
	Hub     string
	HubSet  bool
}

// ResolveWorkspace resolves the working directory, config file path, env file path,
// and git-relative paths based on the provided options.
func ResolveWorkspace(opts WorkspaceOptions) (*WorkspacePaths, error) {
	rawWorkDir := opts.WorkDir
	if rawWorkDir == "" {
		rawWorkDir = "."
	}
	cwd, err := filepath.Abs(rawWorkDir)
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	var resolvedWorkDir string

	if opts.HubSet {
		if strings.TrimSpace(opts.Hub) == "" {
			return nil, fmt.Errorf("hub path cannot be empty")
		}
		if hasParentDirTraversal(opts.Hub) {
			return nil, fmt.Errorf("hub path %q cannot contain '..'", opts.Hub)
		}
		if filepath.IsAbs(opts.Hub) {
			resolvedWorkDir = filepath.Clean(opts.Hub)
		} else {
			resolvedWorkDir = filepath.Join(cwd, opts.Hub)
		}

		rel, err := filepath.Rel(cwd, resolvedWorkDir)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("hub path %q resolves outside current working directory", opts.Hub)
		}
		if rel == "." {
			return nil, fmt.Errorf("hub path %q must resolve to a directory below current working directory", opts.Hub)
		}
	} else {
		resolvedWorkDir = cwd
	}

	resolvedConfigFile, err := GetFullPath("config.yaml", resolvedWorkDir)
	if err != nil {
		return nil, fmt.Errorf("get config file path: %w", err)
	}

	gitRoot, err := FindGitRepoRoot(resolvedWorkDir)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	gitRelPath := ComputeGitRelativePath(gitRoot, resolvedWorkDir)

	resolvedEnvFile := filepath.Join(resolvedWorkDir, ".env")
	if _, err := os.Stat(resolvedEnvFile); os.IsNotExist(err) && gitRoot != "" {
		rootEnv := filepath.Join(gitRoot, ".env")
		if _, err := os.Stat(rootEnv); err == nil {
			resolvedEnvFile = rootEnv
		}
	}

	return &WorkspacePaths{
		WorkDir:        resolvedWorkDir,
		ConfigFilePath: resolvedConfigFile,
		EnvFilePath:    resolvedEnvFile,
		GitRoot:        gitRoot,
		GitRelPath:     gitRelPath,
	}, nil
}

// ResolveWorkspaceFromCommand extracts options from a cli.Command and resolves workspace paths.
func ResolveWorkspaceFromCommand(cmd *cli.Command) (*WorkspacePaths, error) {
	opts := WorkspaceOptions{}
	if cmd.IsSet("hub") {
		// When "hub" is configured as a StringSliceFlag (e.g. generate subcommand)
		if hubs := cmd.StringSlice("hub"); len(hubs) > 0 {
			opts.Hub = hubs[0]
			opts.HubSet = true
		} else if hubStr := cmd.String("hub"); hubStr != "" {
			opts.Hub = hubStr
			opts.HubSet = true
		}
	}
	return ResolveWorkspace(opts)
}

func hasParentDirTraversal(p string) bool {
	cleaned := filepath.ToSlash(p)
	for _, part := range strings.Split(cleaned, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}
