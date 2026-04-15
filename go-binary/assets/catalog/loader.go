package catalog

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	goYaml "go.yaml.in/yaml/v3"
)

const servicesDirectory = "services"

//go:embed services/*.yaml
var embeddedServicesFS embed.FS

type LoadOptions struct {
	DistributionPath string
	Overwrite        bool
}

func LoadBuiltIn() (Catalog, error) {
	return loadFromFS(embeddedServicesFS, servicesDirectory)
}

func Load(options LoadOptions) (Catalog, error) {
	builtIn, err := LoadBuiltIn()
	if err != nil {
		return Catalog{}, err
	}

	if strings.TrimSpace(options.DistributionPath) == "" {
		return builtIn, nil
	}

	externalRoot, err := resolveExternalServicesPath(options.DistributionPath)
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

func resolveExternalServicesPath(distributionPath string) (string, error) {
	cleaned := filepath.Clean(distributionPath)

	servicesDir := filepath.Join(cleaned, servicesDirectory)
	servicesInfo, err := os.Stat(servicesDir)
	if err == nil {
		if !servicesInfo.IsDir() {
			return "", fmt.Errorf("distribution services path %q is not a directory", servicesDir)
		}
		return servicesDir, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("stat distribution services path %q: %w", servicesDir, err)
	}

	rootInfo, err := os.Stat(cleaned)
	if err != nil {
		return "", fmt.Errorf("stat distribution path %q: %w", cleaned, err)
	}
	if !rootInfo.IsDir() {
		return "", fmt.Errorf("distribution path %q is not a directory", cleaned)
	}

	return cleaned, nil
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
		if !strings.HasSuffix(strings.ToLower(path), ".yaml") {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return Catalog{}, fmt.Errorf("walk service definitions: %w", err)
	}

	sort.Strings(files)
	for _, path := range files {
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return Catalog{}, fmt.Errorf("read %q: %w", path, err)
		}

		var definition ServiceDefinition
		if err := goYaml.Unmarshal(content, &definition); err != nil {
			return Catalog{}, fmt.Errorf("unmarshal %q: %w", path, err)
		}
		if err := definition.Validate(); err != nil {
			return Catalog{}, fmt.Errorf("invalid service definition %q: %w", path, err)
		}

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
