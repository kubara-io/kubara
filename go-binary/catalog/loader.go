package catalog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const servicesDirectory = "services"
const builtInServicesDirectory = builtInRootDirectory + "/" + servicesDirectory

type LoadOptions struct {
	CatalogPath string
	Overwrite   bool
}

func LoadBuiltIn() (Catalog, error) {
	return loadFromFS(BuiltInFS(), builtInServicesDirectory)
}

func Load(options LoadOptions) (Catalog, error) {
	builtIn, err := LoadBuiltIn()
	if err != nil {
		return Catalog{}, err
	}

	if strings.TrimSpace(options.CatalogPath) == "" {
		return builtIn, nil
	}

	externalRoot, err := resolveServicesPath(options.CatalogPath)
	if err != nil {
		return Catalog{}, err
	}

	external, err := loadFromFS(os.DirFS(externalRoot), ".")
	if err != nil {
		return Catalog{}, err
	}

	merged := builtIn.Clone()
	for name, def := range external.Services {
		if _, exists := merged.Services[name]; exists && !options.Overwrite {
			return Catalog{}, fmt.Errorf("service definition %q already exists in built-in catalog", name)
		}
		merged.Services[name] = def
	}

	return merged, nil
}

func resolveServicesPath(catalogPath string) (string, error) {
	cleaned := filepath.Clean(catalogPath)

	rootInfo, err := os.Stat(cleaned)
	if err != nil {
		return "", fmt.Errorf("catalog directory %q does not exist: %w", cleaned, err)
	}
	if !rootInfo.IsDir() {
		return "", fmt.Errorf("catalog path %q is not a directory", cleaned)
	}

	servicesDir := filepath.Join(cleaned, servicesDirectory)
	servicesInfo, err := os.Stat(servicesDir)
	if err == nil {
		if !servicesInfo.IsDir() {
			return "", fmt.Errorf("catalog services path %q is not a directory", servicesDir)
		}
		return servicesDir, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return cleaned, nil
	}
	return "", fmt.Errorf("failed to stat catalog services path %q: %w", servicesDir, err)
}

func loadFromFS(fsys fs.FS, root string) (Catalog, error) {
	catalog := Catalog{Services: map[string]ServiceDefinition{}}

	var files []string
	if err := fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		lowerPath := strings.ToLower(path)
		if !strings.HasSuffix(lowerPath, ".yaml") && !strings.HasSuffix(lowerPath, ".yml") {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return Catalog{}, fmt.Errorf("failed to walk service definitions: %w", err)
	}

	sort.Strings(files)
	for _, path := range files {
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return Catalog{}, fmt.Errorf("failed to read %q: %w", path, err)
		}

		var definition ServiceDefinition
		if err := yaml.Unmarshal(content, &definition); err != nil {
			return Catalog{}, fmt.Errorf("failed to unmarshal %q: %w", path, err)
		}
		if err := definition.Validate(); err != nil {
			return Catalog{}, fmt.Errorf("invalid service definition %q: %w", path, err)
		}

		// Canonicalize legacy aliases (for example camelCase names) to the
		// canonical kebab-case service name.
		canonicalName := CanonicalServiceName(definition.Metadata.Name)
		if canonicalName != definition.Metadata.Name {
			definition.Metadata.Name = canonicalName
		}

		if _, exists := catalog.Services[definition.Metadata.Name]; exists {
			return Catalog{}, fmt.Errorf("duplicate service definition %q in %q", definition.Metadata.Name, path)
		}
		catalog.Services[definition.Metadata.Name] = definition
	}

	return catalog, nil
}
