package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const maxInfrastructureOutputSize = 1024 * 1024

type infrastructureOutputReader interface {
	Read(ctx context.Context, dir, command, name string) (string, error)
}

type commandInfrastructureOutputReader struct{}

type cappedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	originalLength := len(p)
	remaining := b.limit - b.Len()
	if remaining <= 0 {
		b.exceeded = true
		return originalLength, nil
	}
	if len(p) > remaining {
		p = p[:remaining]
		b.exceeded = true
	}
	_, _ = b.Buffer.Write(p)
	return originalLength, nil
}

func (commandInfrastructureOutputReader) Read(ctx context.Context, dir, command, name string) (string, error) {
	cmd := exec.CommandContext(ctx, command, "output", "-raw", name)
	cmd.Dir = dir

	stdout := cappedBuffer{limit: maxInfrastructureOutputSize}
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s output %q failed: %w; ensure the infrastructure has been applied and the output exists", command, name, err)
	}
	if stdout.exceeded {
		return "", fmt.Errorf("%s output %q exceeds %d bytes", command, name, maxInfrastructureOutputSize)
	}

	value := strings.TrimSpace(stdout.String())
	if value == "" {
		return "", fmt.Errorf("%s output %q is empty", command, name)
	}
	return value, nil
}

func resolveIaCCommand(configured string) (string, error) {
	switch configured {
	case "terraform", "tofu":
		if _, err := exec.LookPath(configured); err != nil {
			return "", fmt.Errorf("%s executable not found: %w", configured, err)
		}
		return configured, nil
	case "", "auto":
		_, terraformErr := exec.LookPath("terraform")
		_, tofuErr := exec.LookPath("tofu")
		switch {
		case terraformErr == nil && tofuErr != nil:
			return "terraform", nil
		case tofuErr == nil && terraformErr != nil:
			return "tofu", nil
		case terraformErr == nil && tofuErr == nil:
			return "", fmt.Errorf("both terraform and tofu are installed; select one with --iac-command")
		default:
			return "", fmt.Errorf("neither terraform nor tofu executable was found")
		}
	default:
		return "", fmt.Errorf("unsupported --iac-command %q; expected auto, terraform, or tofu", configured)
	}
}
