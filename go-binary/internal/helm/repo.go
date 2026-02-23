package helm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// RepoOptions for helm repository operations
type RepoOptions struct {
	Name     string
	URL      string
	Username string
	Password string
	CertFile string
	KeyFile  string
	CAFile   string
	Timeout  time.Duration
}

// AddRepository adds a helm repository
func AddRepository(ctx context.Context, opts RepoOptions) error {
	args := []string{"repo", "add", opts.Name, opts.URL}

	if opts.Username != "" && opts.Password != "" {
		args = append(args, "--username", opts.Username, "--password", opts.Password)
	}

	if opts.CertFile != "" && opts.KeyFile != "" {
		args = append(args, "--cert-file", opts.CertFile, "--key-file", opts.KeyFile)
	}

	if opts.CAFile != "" {
		args = append(args, "--ca-file", opts.CAFile)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return &HelmRepoError{
			Operation: "add",
			RepoName:  opts.Name,
			Err:       err,
			Stderr:    stderr.String(),
		}
	}

	return nil
}

// UpdateRepository updates a helm repository
func UpdateRepository(name string, timeout time.Duration) error {
	args := []string{"repo", "update"}
	if name != "" {
		args = append(args, name)
	}

	cmd := exec.Command("helm", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, "helm", args...)
		cmd.Stderr = &stderr
	}

	err := cmd.Run()
	if err != nil {
		return &HelmRepoError{
			Operation: "update",
			RepoName:  name,
			Err:       err,
			Stderr:    stderr.String(),
		}
	}

	return nil
}

// Repository represents a helm repository
type Repository struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// HelmRepoError provides detailed error information for helm repo operations
type HelmRepoError struct {
	Operation string
	RepoName  string
	Err       error
	Stderr    string
}

func (e *HelmRepoError) Error() string {
	if e.RepoName != "" {
		return fmt.Sprintf("helm repo %s failed for %s: %v\nStderr: %s",
			e.Operation, e.RepoName, e.Err, e.Stderr)
	}
	return fmt.Sprintf("helm repo %s failed: %v\nStderr: %s",
		e.Operation, e.Err, e.Stderr)
}

func (e *HelmRepoError) Unwrap() error {
	return e.Err
}
