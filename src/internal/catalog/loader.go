package catalog

import (
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

type LoadOptions struct {
	BootstrapCatalog string
	Catalogs         []string
	Overwrite        bool
}

func Load(options LoadOptions) (Catalog, error) {
	merged := Catalog{
		Services: make(map[string]ServiceDefinition),
	}

	bootstrapCatalog := strings.TrimSpace(options.BootstrapCatalog)
	if bootstrapCatalog == "" {
		bootstrapCatalog = DefaultBootstrapCatalog
	}

	bootstrap, err := loadCatalogSource(bootstrapCatalog)
	if err != nil {
		return Catalog{}, fmt.Errorf("load bootstrap catalog: %w", err)
	}

	for name, def := range bootstrap.Services {
		merged.Services[name] = def
	}

	for _, cat := range options.Catalogs {
		if strings.TrimSpace(cat) == "" {
			return Catalog{}, fmt.Errorf("catalog source is empty")
		}

		external, err := loadCatalogSource(cat)
		if err != nil {
			return Catalog{}, fmt.Errorf("load catalog %q: %w", cat, err)
		}

		for name, def := range external.Services {
			if _, exists := merged.Services[name]; exists && !options.Overwrite {
				return Catalog{}, fmt.Errorf("service definition %q already exists in another catalog", name)
			}
			merged.Services[name] = def
		}
	}

	return merged, nil
}

func loadCatalogSource(reference string) (Catalog, error) {
	source, err := ResolveSource(reference)
	if err != nil {
		return Catalog{}, fmt.Errorf("resolve catalog source: %w", err)
	}

	loaded, err := loadFromFS(os.DirFS(source.ServicesPath), ".")
	if err != nil {
		return Catalog{}, fmt.Errorf("load catalog from %q: %w", source.ServicesPath, err)
	}

	return loaded, nil
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
		return Catalog{}, fmt.Errorf("walk service definitions: %w", err)
	}

	sort.Strings(files)
	for _, path := range files {
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return Catalog{}, fmt.Errorf("read %q: %w", path, err)
		}

		var definition ServiceDefinition
		if err := yaml.Unmarshal(content, &definition); err != nil {
			return Catalog{}, fmt.Errorf("unmarshal %q: %w", path, err)
		}
		if err := definition.Validate(); err != nil {
			return Catalog{}, fmt.Errorf("invalid service definition %q: %w", path, err)
		}

		if _, exists := catalog.Services[definition.Metadata.Name]; exists {
			return Catalog{}, fmt.Errorf("duplicate service definition %q in %q", definition.Metadata.Name, path)
		}
		catalog.Services[definition.Metadata.Name] = definition
	}

	return catalog, nil
}
