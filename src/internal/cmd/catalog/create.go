package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	catalogTypes "github.com/kubara-io/kubara/internal/catalog"
	"github.com/rs/zerolog/log"
)

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

	catalogYaml := fmt.Sprintf(`apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: %s`, catalogName)

	if err = os.WriteFile(filepath.Join(catalogName, "Catalog.yaml"), []byte(catalogYaml), 0o600); err != nil {
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

	if err := os.MkdirAll(filepath.Join(base, "customer-service-catalog", "helm", "example"), 0o755); err != nil {
		return true, fmt.Errorf("cannot create customer-service-catalog helm directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(base, "customer-service-catalog", "terraform", "example"), 0o755); err != nil {
		return true, fmt.Errorf("cannot create customer-service-catalog terraform directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(base, "managed-service-catalog", "helm"), 0o755); err != nil {
		return true, fmt.Errorf("cannot create managed-service-catalog helm directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(base, "managed-service-catalog", "terraform"), 0o755); err != nil {
		return true, fmt.Errorf("cannot create managed-service-catalog terraform directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(base, "services"), 0o755); err != nil {
		return true, fmt.Errorf("cannot create service definition directory: %w", err)
	}

	return true, nil
}
