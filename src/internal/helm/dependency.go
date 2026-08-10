package helm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DependencyOptions for helm dependency operations
type DependencyOptions struct {
	ChartPath   string
	Timeout     time.Duration
	SkipRefresh bool
}

// BuildDependencies builds helm dependencies for a chart.
func BuildDependencies(ctx context.Context, opts DependencyOptions) error {
	return buildDependencies(ctx, opts, runDependencyCommand)
}

type dependencyCommandRunner func(context.Context, []string) (string, error)

func buildDependencies(ctx context.Context, opts DependencyOptions, run dependencyCommandRunner) error {
	operation := "build"
	args := dependencyArgs(operation, opts)
	stderr, err := run(ctx, args)
	if err != nil && isDependencyLockMismatch(stderr) {
		operation = "update"
		args = dependencyArgs(operation, opts)
		stderr, err = run(ctx, args)
	}
	if err != nil {
		return &HelmDependencyError{
			Operation: operation,
			ChartPath: opts.ChartPath,
			Err:       err,
			Stderr:    stderr,
		}
	}

	return nil
}

func dependencyArgs(operation string, opts DependencyOptions) []string {
	args := []string{"dependency", operation}

	if opts.SkipRefresh {
		args = append(args, "--skip-refresh")
	}

	if opts.ChartPath != "" {
		args = append(args, opts.ChartPath)
	}
	return args
}

func runDependencyCommand(ctx context.Context, args []string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stderr.String(), err
}

func isDependencyLockMismatch(stderr string) bool {
	normalized := strings.ToLower(stderr)
	return strings.Contains(normalized, "chart.lock") &&
		(strings.Contains(normalized, "out of sync") || strings.Contains(normalized, "update the dependencies"))
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
