package cmd

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

//go:embed templates/.github/workflows/kubara-generate.yml
var githubActionWorkflow []byte

func NewGitHubActionCmd() *cli.Command {
	return &cli.Command{
		Name:        "github-action",
		Usage:       "Create a GitHub Actions workflow for generation on pull requests",
		UsageText:   "kubara github-action [--overwrite]",
		Description: "Writes .github/workflows/kubara-generate.yml into the working directory. Existing files are preserved unless --overwrite is set. Add local catalog paths to the workflow's path filter before committing it.",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "overwrite", Usage: "Overwrite an existing kubara-generate.yml"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			cwd, err := filepath.Abs(cmd.String("work-dir"))
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			path := filepath.Join(cwd, ".github", "workflows", "kubara-generate.yml")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("create workflow directory: %w", err)
			}
			flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
			if cmd.Bool("overwrite") {
				if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
					return fmt.Errorf("refusing to overwrite non-regular file: %s", path)
				} else if err != nil && !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("inspect workflow file: %w", err)
				}
				flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
			}
			file, err := os.OpenFile(path, flags, 0o644)
			if errors.Is(err, os.ErrExist) {
				log.Info().Str("file", path).Msg("workflow exists, skipping (use --overwrite to refresh)")
				return nil
			}
			if err != nil {
				return fmt.Errorf("create workflow file: %w", err)
			}
			_, writeErr := file.Write(githubActionWorkflow)
			closeErr := file.Close()
			if err := errors.Join(writeErr, closeErr); err != nil {
				return fmt.Errorf("write workflow file: %w", err)
			}
			log.Info().Str("file", path).Msg("wrote GitHub Actions workflow")
			return nil
		},
	}
}
