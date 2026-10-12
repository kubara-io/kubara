package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/cmd/generate"
	"github.com/kubara-io/kubara/internal/render"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

type GenerateFlags struct {
	Terraform bool
	Helm      bool
	DryRun    bool
	All       bool
	Hubs      []string
}

func NewGenerateFlags() *GenerateFlags {
	return &GenerateFlags{
		Terraform: false,
		Helm:      false,
		DryRun:    false,
		All:       false,
		Hubs:      nil,
	}
}

// NewGenerateCmd returns the command with flags added
// TODO implement deep-merge and/or --reset flag
func NewGenerateCmd() *cli.Command {
	flags := NewGenerateFlags()

	cmd := &cli.Command{
		Name:        "generate",
		Usage:       "Generate files from catalog templates",
		UsageText:   "kubara generate [--all|--hubs HUB1,HUB2,...|--hub HUB3] [--terraform|--helm] [--catalog PATH_OR_OCI] [--catalog-overwrite]] [--dry-run]",
		Description: "Renders Helm and Terraform templates from configured local or OCI catalogs for the specified hubs.\nBy default, it generates both template types and targets the hub in the current working directory.",
		Action: func(c context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() > 0 {
				return fmt.Errorf("unexpected positional argument(s): %v", cmd.Args().Slice())
			}
			if flags.All && len(flags.Hubs) > 0 {
				return fmt.Errorf("cannot specify both --all and --hub")
			}
			if flags.All {
				return flags.runAll(cmd)
			}
			if len(flags.Hubs) > 0 {
				return flags.runHubs(cmd)
			}

			o, err := flags.ToOptions(cmd)
			if err != nil {
				return fmt.Errorf("convert flags to options: %w", err)
			}
			return o.Run()
		},
	}

	flags.AddFlags(cmd)

	return cmd
}

func (flags *GenerateFlags) ToOptions(cmd *cli.Command) (*generate.Options, error) {
	ws, err := ResolveWorkspace(cmd)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace: %w", err)
	}

	platformComponents, err := utils.GetFullPath(render.DefaultPlatformComponentsPath, ws.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("get platform-components path: %w", err)
	}

	platformConfigs, err := utils.GetFullPath(render.DefaultPlatformConfigsPath, ws.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("get platform-configs path: %w", err)
	}

	catalogOptions, err := catalogLoadOptionsFromCommand(cmd, "")
	if err != nil {
		return nil, fmt.Errorf("get catalog options: %w", err)
	}

	o := &generate.Options{
		TemplateType:       render.All,
		DryRun:             flags.DryRun,
		CWD:                ws.WorkDir,
		ConfigFilePath:     ws.ConfigFilePath,
		Catalogs:           catalogOptions.Catalogs,
		CatalogOverwrite:   catalogOptions.Overwrite,
		PlatformComponents: platformComponents,
		PlatformConfigs:    platformConfigs,
		EnvPath:            ws.EnvFilePath,
		GitRelPath:         ws.GitRelPath,
	}

	if flags.Helm && !flags.Terraform {
		o.TemplateType = render.Helm
	} else if flags.Terraform && !flags.Helm {
		o.TemplateType = render.Terraform
	}

	return o, nil
}

func (flags *GenerateFlags) generateWorkspace(cmd *cli.Command, wsPaths *utils.WorkspacePaths, workspaceRelPath string) error {
	catalogOpts, err := catalog.ResolveLoadOptions(wsPaths.WorkDir, "", cmd.StringSlice("catalog"), cmd.Bool("catalog-overwrite"))
	if err != nil {
		return fmt.Errorf("failed resolving catalogs for [%s]: %w", workspaceRelPath, err)
	}

	platformComponents := filepath.Join(wsPaths.WorkDir, render.DefaultPlatformComponentsPath)
	platformConfigs := filepath.Join(wsPaths.WorkDir, render.DefaultPlatformConfigsPath)

	tplType := render.All
	if flags.Helm && !flags.Terraform {
		tplType = render.Helm
	} else if flags.Terraform && !flags.Helm {
		tplType = render.Terraform
	}

	opts := &generate.Options{
		TemplateType:       tplType,
		DryRun:             flags.DryRun,
		CWD:                wsPaths.WorkDir,
		ConfigFilePath:     wsPaths.ConfigFilePath,
		Catalogs:           catalogOpts.Catalogs,
		CatalogOverwrite:   catalogOpts.Overwrite,
		PlatformComponents: platformComponents,
		PlatformConfigs:    platformConfigs,
		EnvPath:            wsPaths.EnvFilePath,
		GitRelPath:         wsPaths.GitRelPath,
	}

	return opts.Run()
}

func (flags *GenerateFlags) runHubs(cmd *cli.Command) error {
	var validHubs []string
	for _, hub := range flags.Hubs {
		trimmed := strings.TrimSpace(hub)
		if trimmed != "" {
			validHubs = append(validHubs, trimmed)
		}
	}
	if len(validHubs) == 0 {
		return fmt.Errorf("no valid hub specified in --hub")
	}

	var failed []string
	for _, hub := range validHubs {
		log.Info().Msgf("Generating workspace [%s]...", hub)

		wsPaths, err := utils.ResolveWorkspace(utils.WorkspaceOptions{
			Hub:    hub,
			HubSet: true,
		})
		if err != nil {
			log.Error().Err(err).Msgf("Failed resolving workspace [%s]", hub)
			failed = append(failed, hub)
			continue
		}

		if err := flags.generateWorkspace(cmd, wsPaths, hub); err != nil {
			log.Error().Err(err).Msgf("Failed generating workspace [%s]", hub)
			failed = append(failed, hub)
			continue
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("generation failed for workspace(s): %s", strings.Join(failed, ", "))
	}

	return nil
}

func (flags *GenerateFlags) runAll(cmd *cli.Command) error {
	ws, err := ResolveWorkspace(cmd)
	if err != nil {
		return fmt.Errorf("resolve workspace: %w", err)
	}

	searchRoot := ws.WorkDir
	configs, err := discoverWorkspaces(searchRoot)
	if err != nil {
		return fmt.Errorf("discover workspaces in %q: %w", searchRoot, err)
	}

	if len(configs) == 0 {
		return fmt.Errorf("no config files found in %q", searchRoot)
	}

	log.Info().Msgf("Discovered %d workspace(s) in %s", len(configs), searchRoot)

	var failed []string
	for _, cfgPath := range configs {
		workspaceDir := filepath.Dir(cfgPath)
		workspaceRelPath, _ := filepath.Rel(searchRoot, workspaceDir)
		if workspaceRelPath == "." || workspaceRelPath == "" {
			workspaceRelPath = "(root)"
		}
		log.Info().Msgf("Generating workspace [%s]...", workspaceRelPath)

		wsPaths, err := utils.ResolveWorkspace(utils.WorkspaceOptions{
			WorkDir: workspaceDir,
		})
		if err != nil {
			log.Error().Err(err).Msgf("Failed resolving workspace [%s]", workspaceRelPath)
			failed = append(failed, workspaceRelPath)
			continue
		}

		if err := flags.generateWorkspace(cmd, wsPaths, workspaceRelPath); err != nil {
			log.Error().Err(err).Msgf("Failed generating workspace [%s]", workspaceRelPath)
			failed = append(failed, workspaceRelPath)
			continue
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("generation failed for workspace(s): %s", strings.Join(failed, ", "))
	}

	return nil
}

func discoverWorkspaces(root string) ([]string, error) {
	var configFiles []string
	skipDirs := map[string]bool{
		".git":                true,
		".github":             true,
		"platform-components": true,
		"platform-configs":    true,
		".cache":              true,
		".kubara":             true,
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if skipDirs[name] {
				return filepath.SkipDir
			}
			return nil
		}

		if d.Name() == "config.yaml" || d.Name() == "config.yml" {
			configFiles = append(configFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(configFiles)
	return configFiles, nil
}

func (flags *GenerateFlags) AddFlags(cmd *cli.Command) {
	generateFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:        "all",
			Aliases:     []string{"A"},
			Usage:       "Discover and target all hubs in the working directory",
			Value:       flags.All,
			Destination: &flags.All,
		},
		&cli.StringSliceFlag{
			Name:        "hub",
			Aliases:     []string{"hubs"},
			Usage:       "Target a list of comma separated hub directories",
			Value:       flags.Hubs,
			Destination: &flags.Hubs,
			Config: cli.StringConfig{
				TrimSpace: true,
			},
		},
		&cli.BoolFlag{
			Name:        "terraform",
			Usage:       "Only generate Terraform files",
			Value:       flags.Terraform,
			Destination: &flags.Terraform,
		},
		&cli.BoolFlag{
			Name:        "helm",
			Usage:       "Only generate Helm files",
			Value:       flags.Helm,
			Destination: &flags.Helm,
		},
		&cli.BoolFlag{
			Name:        "dry-run",
			Usage:       "Preview generation without creating files",
			Value:       flags.DryRun,
			Destination: &flags.DryRun,
		},
	}

	cmd.Flags = append(cmd.Flags, generateFlags...)
}
