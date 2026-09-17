package catalog

import (
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

type LoadOptions struct {
	CWD              string
	BootstrapCatalog string
	Catalogs         []string
	Overwrite        bool
}

func Load(options LoadOptions) (Catalog, error) {
	merged := Catalog{
		Services: make(map[string]ServiceDefinition),
	}

	sources, err := ResolveSources(options)
	if err != nil {
		return Catalog{}, err
	}

	bootstrap, err := loadCatalogSource(sources[0])
	if err != nil {
		return Catalog{}, fmt.Errorf("load bootstrap catalog: %w", err)
	}

	for name, def := range bootstrap.Services {
		merged.Services[name] = def
	}

	for _, cat := range sources[1:] {
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
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return Catalog{}, fmt.Errorf("read %q: %w", path, err)
		}

		def, err := DecodeServiceDefinition(data)
		if err != nil {
			return Catalog{}, fmt.Errorf("invalid service definition %q: %w", path, err)
		}

		if _, exists := catalog.Services[def.Name]; exists {
			return Catalog{}, fmt.Errorf("duplicate service definition %q in %q", def.Name, path)
		}
		catalog.Services[def.Name] = def
	}

	return catalog, nil
}
