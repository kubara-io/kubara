package helm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// TemplateOptions for helm template operations
type TemplateOptions struct {
	ReleaseName string
	ChartPath   string
	ValuesPaths []string
	Namespace   string
	SetArgs     []string
	APIVersions []string
	IncludeCRDs bool
	ShowOnly    []string
	KubeVersion string
	Timeout     time.Duration
}

// Template executes helm template with enhanced error handling
func Template(ctx context.Context, opts TemplateOptions) ([]byte, error) {
	args := buildTemplateArgs(opts)

	cmd := exec.CommandContext(ctx, "helm", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, &HelmTemplateError{
			Err:    err,
			Stderr: stderr.String(),
			Args:   args,
		}
	}

	return stdout.Bytes(), nil
}

// buildTemplateArgs constructs helm template arguments
func buildTemplateArgs(opts TemplateOptions) []string {
	args := []string{
		"template",
		opts.ReleaseName,
		opts.ChartPath,
	}

	// Add all values override files to arg
	for _, values := range opts.ValuesPaths {
		args = append(args, "--values", values)
	}

	if opts.Namespace != "" {
		args = append(args, "--namespace", opts.Namespace)
	}

	if opts.IncludeCRDs {
		args = append(args, "--include-crds")
	}

	for _, version := range opts.APIVersions {
		args = append(args, "--api-versions", version)
	}

	for _, set := range opts.SetArgs {
		args = append(args, "--set", set)
	}

	for _, showOnly := range opts.ShowOnly {
		args = append(args, "--show-only", showOnly)
	}

	if opts.KubeVersion != "" {
		args = append(args, "--kube-version", opts.KubeVersion)
	}

	return args
}

// HelmTemplateError provides detailed error information for helm template failures
type HelmTemplateError struct {
	Err    error
	Stderr string
	Args   []string
}

func (e *HelmTemplateError) Error() string {
	return fmt.Sprintf("helm template failed: %v\nStderr: %s\nArgs: %s",
		e.Err, e.Stderr, strings.Join(e.Args, " "))
}

func (e *HelmTemplateError) Unwrap() error {
	return e.Err
}
