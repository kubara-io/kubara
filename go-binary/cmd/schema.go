package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/utils"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

type SchemaOptions struct {
	outputFilePath string
	catalogOptions catalog.LoadOptions
}

type SchemaFlags struct {
	OutputFlag string
}

func NewSchemaFlags() *SchemaFlags {
	return &SchemaFlags{
		OutputFlag: "config.schema.json",
	}
}

func NewSchemaCmd() *cli.Command {
	flags := NewSchemaFlags()
	cmd := &cli.Command{
		Name:      "schema",
		Usage:     "Generate JSON schema file for config structure",
		UsageText: "schema [--output] [--catalog <path> [--catalog-overwrite]]",
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
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	outputFilePath, err := utils.GetFullPath(flags.OutputFlag, cwd)
	if err != nil {
		return nil, fmt.Errorf("get output file path: %w", err)
	}

	catalogOptions, err := catalogLoadOptionsFromCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("get catalog options: %w", err)
	}

	o := &SchemaOptions{
		outputFilePath: outputFilePath,
		catalogOptions: catalogOptions,
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
	}

	cmd.Flags = schemaFlags
}

func (o *SchemaOptions) Run() error {
	// Generate schema
	schemaDoc, err := config.GenerateSchemaWithCatalog(o.catalogOptions)
	if err != nil {
		return fmt.Errorf("generate schema: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(o.outputFilePath), 0750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Marshal to JSON
	schemaJSON, err := json.MarshalIndent(schemaDoc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schema to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(o.outputFilePath, schemaJSON, 0600); err != nil {
		return fmt.Errorf("write schema file: %w", err)
	}

	log.Info().Msgf("Generated schema file: %s", o.outputFilePath)
	return nil
}
