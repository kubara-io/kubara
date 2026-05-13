package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/kubara-io/kubara/internal/generate"
	"github.com/kubara-io/kubara/internal/render"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/urfave/cli/v3"
)

type GenerateFlags struct {
	Terraform          bool
	Helm               bool
	DryRun             bool
	ManagedCatalogPath string
	OverlayValuesPath  string
}

func NewGenerateFlags() *GenerateFlags {
	return &GenerateFlags{
		Terraform:          false,
		Helm:               false,
		DryRun:             false,
		ManagedCatalogPath: render.DefaultManagedCatalogPath,
		OverlayValuesPath:  render.DefaultOverlayValuesPath,
	}
}

// NewGenerateCmd returns the command with flags added
// TODO implement deep-merge and/or --reset flag
func NewGenerateCmd() *cli.Command {
	flags := NewGenerateFlags()

	cmd := &cli.Command{
		Name:        "generate",
		Usage:       "generates files from embedded templates and the config file; by default for both Helm and Terraform",
		UsageText:   "generate [--terraform|--helm] [--managed-catalog <path> --overlay-values <path>] [--catalog <path> [--catalog-overwrite]] [--dry-run]",
		Description: "generate reads config values and templates the embedded Helm and Terraform files.",
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
	managedCatalogPath, err := utils.GetFullPath(cmd.String("managed-catalog"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get managed catalog path: %w", err)
	}
	overlayValuesPath, err := utils.GetFullPath(cmd.String("overlay-values"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get overlay values path: %w", err)
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
		CatalogPath:        catalogOptions.CatalogPath,
		CatalogOverwrite:   catalogOptions.Overwrite,
		ManagedCatalogPath: managedCatalogPath,
		OverlayValuesPath:  overlayValuesPath,
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
		&cli.StringFlag{
			Name:        "managed-catalog",
			Usage:       "Path to the managed catalog directory.",
			Value:       render.DefaultManagedCatalogPath,
			Destination: &flags.ManagedCatalogPath,
		},
		&cli.StringFlag{
			Name:        "overlay-values",
			Usage:       "Path to overlay values directory.",
			Value:       render.DefaultOverlayValuesPath,
			Destination: &flags.OverlayValuesPath,
		},
	}

	cmd.Flags = append(cmd.Flags, generateFlags...)
}
