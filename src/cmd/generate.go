package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/kubara-io/kubara/internal/cmd/generate"
	"github.com/kubara-io/kubara/internal/render"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/urfave/cli/v3"
)

type GenerateFlags struct {
	Terraform bool
	Helm      bool
	DryRun    bool
}

func NewGenerateFlags() *GenerateFlags {
	return &GenerateFlags{
		Terraform: false,
		Helm:      false,
		DryRun:    false,
	}
}

// NewGenerateCmd returns the command with flags added
// TODO implement deep-merge and/or --reset flag
func NewGenerateCmd() *cli.Command {
	flags := NewGenerateFlags()

	cmd := &cli.Command{
		Name:        "generate",
		Usage:       "Generate files from catalog templates",
		UsageText:   "kubara generate [--terraform|--helm] [--catalog PATH_OR_OCI [--catalog-overwrite]] [--dry-run]",
		Description: "Renders embedded Helm and Terraform templates using values from the config file. By default, it generates both template types.",
		Action: func(c context.Context, cmd *cli.Command) error {
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
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	configFilePath, err := utils.GetFullPath(cmd.String("config-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get config file path: %w", err)
	}
	platformComponents, err := utils.GetFullPath(render.DefaultPlatformComponentsPath, cwd)
	if err != nil {
		return nil, fmt.Errorf("get platform-components path: %w", err)
	}
	platformConfigs, err := utils.GetFullPath(render.DefaultPlatformConfigsPath, cwd)
	if err != nil {
		return nil, fmt.Errorf("get platform-configs path: %w", err)
	}
	catalogOptions, err := catalogLoadOptionsFromCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("get catalog options: %w", err)
	}
	envPath, err := utils.GetFullPath(cmd.String("env-file"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get env path: %w", err)
	}

	o := &generate.Options{
		TemplateType:       render.All,
		DryRun:             flags.DryRun,
		CWD:                cwd,
		ConfigFilePath:     configFilePath,
		Catalogs:           catalogOptions.Catalogs,
		CatalogOverwrite:   catalogOptions.Overwrite,
		PlatformComponents: platformComponents,
		PlatformConfigs:    platformConfigs,
		EnvPath:            envPath,
	}

	if flags.Helm && !flags.Terraform {
		o.TemplateType = render.Helm
	} else if flags.Terraform && !flags.Helm {
		o.TemplateType = render.Terraform
	}

	return o, nil
}

func (flags *GenerateFlags) AddFlags(cmd *cli.Command) {
	generateFlags := []cli.Flag{
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
