package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"kubara/assets/catalog"
	"kubara/assets/config"
	"kubara/utils"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

type SchemaOptions struct {
	outputFilePath  string
	catalogPath     string
	catalogOverride bool
}

type SchemaFlags struct {
	OutputFlag      string
	CatalogPath     string
	CatalogOverride bool
}

func NewSchemaFlags() *SchemaFlags {
	return &SchemaFlags{
		OutputFlag:      "config.schema.json",
		CatalogPath:     "",
		CatalogOverride: false,
	}
}

func NewSchemaCmd() *cli.Command {
	flags := NewSchemaFlags()
	cmd := &cli.Command{
		Name:      "schema",
		Usage:     "Generate JSON schema file for config structure",
		UsageText: "schema [--output] [--catalog <path> [--force|--overwrite]]",
		Action: func(c context.Context, cmd *cli.Command) error {
			o, err := flags.ToOptions(cmd)
			if err != nil {
				return err
			}
			return o.Run()
		},
	}

	flags.AddFlags(cmd)

	return cmd
}

func (flags *SchemaFlags) ToOptions(cmd *cli.Command) (*SchemaOptions, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return nil, err
	}
	outputFilePath, err := utils.GetFullPath(flags.OutputFlag, cwd)
	if err != nil {
		return nil, err
	}
	catalogPath := ""
	if flags.CatalogPath != "" {
		catalogPath, err = utils.GetFullPath(flags.CatalogPath, cwd)
		if err != nil {
			return nil, fmt.Errorf("failed to get catalog path: %w", err)
		}
	}

	o := &SchemaOptions{
		outputFilePath:  outputFilePath,
		catalogPath:     catalogPath,
		catalogOverride: flags.CatalogOverride,
	}
	return o, nil
}

func (flags *SchemaFlags) AddFlags(cmd *cli.Command) {
	schemaFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "output",
			Aliases:     []string{"o"},
			Value:       flags.OutputFlag,
			Usage:       "Output file path for the JSON schema",
			Destination: &flags.OutputFlag,
		},
		&cli.StringFlag{
			Name:        "catalog",
			Value:       flags.CatalogPath,
			Usage:       "Path to external ServiceDefinition catalog directory.",
			Destination: &flags.CatalogPath,
		},
		&cli.BoolFlag{
			Name:        "overwrite",
			Aliases:     []string{"force"},
			Value:       flags.CatalogOverride,
			Usage:       "Allow external service definitions from --catalog to overwrite built-in definitions on name collisions.",
			Destination: &flags.CatalogOverride,
		},
	}

	cmd.Flags = schemaFlags
}

func (o *SchemaOptions) Run() error {
	// Generate schema
	schemaDoc, err := config.GenerateSchemaWithCatalog(catalog.LoadOptions{
		CatalogPath: o.catalogPath,
		Overwrite:   o.catalogOverride,
	})
	if err != nil {
		return fmt.Errorf("failed to generate schema: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(o.outputFilePath), 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to JSON
	schemaJSON, err := json.MarshalIndent(schemaDoc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(o.outputFilePath, schemaJSON, 0600); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	log.Info().Msgf("Generated schema file: %s", o.outputFilePath)
	return nil
}
