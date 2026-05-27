package catalog

import (
	"context"
	"fmt"
	"os"
	"path"

	catalogTypes "github.com/kubara-io/kubara/internal/catalog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func NewCatalogCreate() *cli.Command {
	cmd := &cli.Command{
		Name:        "create",
		Usage:       "Create a custom catalog directory skeleton",
		UsageText:   "kubara catalog create CATALOG_NAME",
		Description: "Scaffolds a custom catalog directory with Catalog.yaml plus customer-service-catalog, managed-service-catalog, and services directories.",
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

			return createCatalog(catalogName)
		},
	}

	return cmd
}

func createCatalog(catalogName string) error {
	if !catalogTypes.RFC1123Label.MatchString(catalogName) {
		return fmt.Errorf("catalog name must adhere to rfc 1123: must be 1-63 characters, start with a lowercase letter, contain only lowercase letters, digits, or '-', and end with a letter or digit")
	}

	if _, err := os.Stat(catalogName); err == nil {
		return fmt.Errorf("a directory with name %s already exists", catalogName)
	}

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

	log.Info().Msg("Catalog has been created")

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
