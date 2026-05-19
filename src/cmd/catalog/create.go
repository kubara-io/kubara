package catalog

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/urfave/cli/v3"
)

func NewCatalogCreate() *cli.Command {
	cmd := &cli.Command{
		Name:        "create",
		Usage:       "create NAME",
		UsageText:   "Creates a new custom catalog directory",
		Description: "...",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "catalog-name",
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
		},
		Action: func(c context.Context, cmd *cli.Command) error {
			catalogName := cmd.StringArg("catalog-name")
			if len(catalogName) == 0 {
				cli.ShowSubcommandHelpAndExit(cmd, 1)
			}

			return creation(catalogName)
		},
	}

	return cmd
}

func creation(catalogName string) error {
	if err := createDirectories(catalogName); err != nil {
		return err
	}

	catalogYaml := fmt.Sprintf(`apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: %s`, catalogName)

	if err := os.WriteFile(path.Join(catalogName, "Catalog.yaml"), []byte(catalogYaml), 0o600); err != nil {
		return fmt.Errorf("cannot create Catalog.yaml: %w", err)
	}

	return nil
}

func createDirectories(base string) error {
	err := os.Mkdir(base, 0o755)
	if err != nil {
		return fmt.Errorf("cannot create catalog directory: %w", err)
	}

	if err := os.MkdirAll(path.Join(base, "customer-service-catalog", "helm", "example"), 0o755); err != nil {
		return fmt.Errorf("cannot create customer-service-catalog helm directory: %w", err)
	}

	if err := os.MkdirAll(path.Join(base, "customer-service-catalog", "terraform", "example"), 0o755); err != nil {
		return fmt.Errorf("cannot create customer-service-catalog terraform directory: %w", err)
	}

	if err := os.MkdirAll(path.Join(base, "managed-service-catalog", "helm"), 0o755); err != nil {
		return fmt.Errorf("cannot create managed-service-catalog helm directory: %w", err)
	}

	if err := os.MkdirAll(path.Join(base, "managed-service-catalog", "terraform"), 0o755); err != nil {
		return fmt.Errorf("cannot create managed-service-catalog terraform directory: %w", err)
	}

	if err := os.MkdirAll(path.Join(base, "services"), 0o755); err != nil {
		return fmt.Errorf("cannot create service definition directory: %w", err)
	}

	return nil
}
