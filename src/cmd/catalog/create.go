package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	catalogTypes "github.com/kubara-io/kubara/internal/catalog"
	"sigs.k8s.io/yaml"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

func NewCatalogCreate() *cli.Command {
	cmd := &cli.Command{
		Name:        "create",
		Usage:       "Create a custom catalog directory skeleton",
		UsageText:   "kubara catalog create CATALOG_NAME",
		Description: "Scaffolds a custom catalog directory with Catalog.yaml plus platform-configs, platform-components, and services directories.",
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

			return CreateCatalog(catalogName)
		},
	}

	return cmd
}
func CreateCatalog(catalogName string) (err error) {
	if !catalogTypes.RFC1123Label.MatchString(catalogName) {
		return fmt.Errorf("catalog name must adhere to rfc 1123: must be 1-63 characters, start with a lowercase letter, contain only lowercase letters, digits, or '-', and end with a letter or digit")
	}

	if _, err := os.Stat(catalogName); err == nil {
		return fmt.Errorf("a directory with name %s already exists", catalogName)
	}

	createdRoot, err := createDirectories(catalogName)
	if err != nil {
		if !createdRoot {
			return err
		}

		return cleanupCatalogRoot(catalogName, err)
	}

	defer func() {
		if err == nil {
			return
		}

		err = cleanupCatalogRoot(catalogName, err)
	}()

	catalogScaffold := catalogTypes.CatalogManifest{
		APIVersion: catalogTypes.CatalogAPIVersion,
		Kind:       catalogTypes.CatalogKind,
		Metadata: catalogTypes.Metadata{
			Name: catalogName,
		},
		Spec: catalogTypes.CatalogSpec{
			Version: "0.1.0",
		},
	}

	catalogYaml, err := yaml.Marshal(catalogScaffold)
	if err != nil {
		return fmt.Errorf("cannot marshal Catalog.yaml: %w", err)
	}

	if err = os.WriteFile(filepath.Join(catalogName, "Catalog.yaml"), catalogYaml, 0o600); err != nil {
		return fmt.Errorf("cannot create Catalog.yaml: %w", err)
	}

	log.Info().Msg("Catalog has been successfully created")

	return nil
}

func cleanupCatalogRoot(catalogName string, err error) error {
	if cleanupErr := os.RemoveAll(catalogName); cleanupErr != nil {
		return fmt.Errorf("%w: cleanup failed: %v", err, cleanupErr)
	}

	return err
}

func createDirectories(base string) (bool, error) {
	err := os.Mkdir(base, 0o755)
	if err != nil {
		return false, fmt.Errorf("cannot create catalog directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(base, "platform-configs", "helm"), 0o755); err != nil {
		return true, fmt.Errorf("cannot create platform-configs helm directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(base, "platform-configs", "terraform"), 0o755); err != nil {
		return true, fmt.Errorf("cannot create platform-configs terraform directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(base, "platform-components", "helm"), 0o755); err != nil {
		return true, fmt.Errorf("cannot create platform-components helm directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(base, "platform-components", "terraform"), 0o755); err != nil {
		return true, fmt.Errorf("cannot create platform-components terraform directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(base, "services"), 0o755); err != nil {
		return true, fmt.Errorf("cannot create service definition directory: %w", err)
	}

	return true, nil
}
