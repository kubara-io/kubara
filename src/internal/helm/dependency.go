package helm

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// DependencyOptions for helm dependency operations
type DependencyOptions struct {
	ChartPath   string
	Timeout     time.Duration
	SkipRefresh bool
}

// // BuildDependencies builds helm dependencies for a chart
// func BuildDependencies(ctx context.Context, opts DependencyOptions) error {
// 	args := []string{"dependency", "build"}

// 	if opts.SkipRefresh {
// 		args = append(args, "--skip-refresh")
// 	}

// 	if opts.ChartPath != "" {
// 		args = append(args, opts.ChartPath)
// 	}

// 	var stdout, stderr bytes.Buffer
// 	cmd := exec.CommandContext(ctx, "helm", args...)
// 	cmd.Stdout = &stdout
// 	cmd.Stderr = &stderr

// 	err := cmd.Run()
// 	if err != nil {
// 		return &HelmDependencyError{
// 			Operation: "build",
// 			ChartPath: opts.ChartPath,
// 			Err:       err,
// 			Stderr:    stderr.String(),
// 		}
// 	}

// 	return nil
// }

// CleanDependencies removes the helm dependency lock file and the previously
// downloaded subchart archives so the next `helm dependency update` re-resolves
// the dependency tree from the current repository indexes. A stale Chart.lock
// referencing a subchart version that is no longer available leads to
// "can't get a valid version for subchart" errors.
func CleanDependencies(chartPath string) error {
	if chartPath == "" {
		return nil
	}

	for _, p := range []string{
		filepath.Join(chartPath, "Chart.lock"),
		filepath.Join(chartPath, "charts"),
	} {
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("clean helm dependency artifact %q: %w", p, err)
		}
	}

	return nil
}

// UpdateDependencies updates helm dependencies for a chart
func UpdateDependencies(ctx context.Context, opts DependencyOptions) error {
	args := []string{"dependency", "update"}
	if opts.ChartPath != "" {
		args = append(args, opts.ChartPath)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return &HelmDependencyError{
			Operation: "update",
			ChartPath: opts.ChartPath,
			Err:       err,
			Stderr:    stderr.String(),
		}
	}

	return nil
}

// Dependency represents a helm chart dependency
type Dependency struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Repository string `json:"repository"`
	Status     string `json:"status"`
}

// HelmDependencyError provides detailed error information for helm dependency operations
type HelmDependencyError struct {
	Operation string
	ChartPath string
	Err       error
	Stderr    string
}

func (e *HelmDependencyError) Error() string {
	return fmt.Sprintf("helm dependency %s failed for %s: %v\nStderr: %s",
		e.Operation, e.ChartPath, e.Err, e.Stderr)
}

func (e *HelmDependencyError) Unwrap() error {
	return e.Err
}
