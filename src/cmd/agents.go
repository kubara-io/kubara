package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kubara-io/kubara/internal/agentcontext"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

type AgentsFlags struct {
	OverwriteFlag bool
}

func NewAgentsFlags() *AgentsFlags {
	return &AgentsFlags{
		OverwriteFlag: false,
	}
}

func NewAgentsCmd() *cli.Command {
	flags := NewAgentsFlags()

	cmd := &cli.Command{
		Name:        "agents",
		Usage:       "Scaffold an onboarding file for AI coding assistants (AGENTS.md)",
		UsageText:   "kubara agents [--overwrite]",
		Description: "Writes AGENTS.md into the working directory so AI coding assistants (Claude Code, Codex, …) have a compact, token-lean entry point into kubara. It delegates command and config details to the self-describing CLI (kubara --help, kubara schema) and links the published Markdown documentation for the installed kubara version on the docs site. The existing file is left untouched unless --overwrite is set. Commit it so it travels with the repository.",
		Action: func(_ context.Context, cmd *cli.Command) error {
			cwd, err := filepath.Abs(cmd.String("work-dir"))
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			return runAgents(cwd, flags.OverwriteFlag)
		},
	}

	cmd.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "overwrite",
			Value:       flags.OverwriteFlag,
			Usage:       "Overwrite an existing AGENTS.md",
			Destination: &flags.OverwriteFlag,
		},
	}

	return cmd
}

func runAgents(cwd string, overwrite bool) error {
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		return fmt.Errorf("create working directory: %w", err)
	}
	result, err := agentcontext.Write(cwd, version, overwrite)
	if err != nil {
		return fmt.Errorf("write agent context file: %w", err)
	}
	if result.Written {
		log.Info().Str("file", result.Path).Msg("✓ wrote agent context file")
	} else {
		log.Info().Str("file", result.Path).Msg("agent context file exists, skipping (use --overwrite to refresh)")
	}
	return nil
}
