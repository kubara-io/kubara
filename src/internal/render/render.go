package render

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/kubara-io/kubara/internal/catalog"

	libtemplate "github.com/kubara-io/libkubara/template"
	"github.com/kubara-io/libkubara/template/tree"
)

type TemplateType int

const (
	Terraform TemplateType = iota
	Helm
	All
)

const (
	DefaultPlatformComponentsPath string = "platform-components"
	DefaultPlatformConfigsPath    string = "platform-configs"
	maxTemplateOutputBytes        int64  = 16 << 20
)

var templateName = map[TemplateType]string{
	Terraform: "terraform",
	Helm:      "helm",
	All:       "all",
}

func TemplateFiles(options TemplateOptions) ([]TemplateResult, error) {
	if err := validateTemplateType(options.Type); err != nil {
		return nil, err
	}
	sources, err := loadTemplateSources(options)
	if err != nil {
		return nil, fmt.Errorf("load template sources for provider %q: %w", options.Provider, err)
	}
	engine, err := libtemplate.New(
		libtemplate.WithHermeticSprig(),
		libtemplate.WithMissingKeyError(),
		libtemplate.WithMaxOutput(maxTemplateOutputBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("create template engine: %w", err)
	}
	treeSources := make([]tree.Source, 0, len(sources))
	for _, source := range sources {
		treeSources = append(treeSources, tree.Source{
			Name:  source.name,
			FS:    source.fsys,
			Roots: templateRootsForType(source.baseRoot, options.Type),
		})
	}
	renderer, err := tree.New(engine,
		tree.WithSources(treeSources...),
		tree.WithPredicate(templateTreePredicate(options)),
		tree.WithPathFunc(func(entry tree.Entry) (string, error) {
			return entry.Path, nil
		}),
		tree.WithKeyFunc(func(entry tree.Entry) (string, error) {
			return StripProviderPath(entry.Path), nil
		}),
		tree.WithCollisionResolver(templateTreeCollisionResolver(options.CatalogOptions.Overwrite)),
	)
	if err != nil {
		return nil, fmt.Errorf("create template tree renderer: %w", err)
	}
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	rendered, renderErr := renderer.RenderAll(ctx, options.Data)
	results := make([]TemplateResult, 0, len(rendered))
	for _, result := range rendered {
		results = append(results, TemplateResult{
			Path:    result.Path,
			Content: string(result.Content),
			Error:   result.Error,
		})
	}
	return results, renderErr
}

func templateTreePredicate(options TemplateOptions) tree.Predicate {
	selectedProvider := normalizeProviderName(options.Provider)
	return func(entry tree.Entry) bool {
		_, provider, providerSpecific := splitProviderPath(entry.Path)
		if providerSpecific && provider != selectedProvider {
			return false
		}
		return options.PathPredicate == nil || options.PathPredicate(StripProviderPath(entry.Path))
	}
}

func templateTreeCollisionResolver(overwrite bool) tree.CollisionResolver {
	return func(current, next tree.Entry) (bool, error) {
		strippedPath := StripProviderPath(next.Path)
		if current.SourceIndex != next.SourceIndex {
			if !overwrite {
				return false, fmt.Errorf("template %q already exists in both %q and %q", strippedPath, current.Source, next.Source)
			}
			return next.SourceIndex > current.SourceIndex, nil
		}
		_, _, currentProviderSpecific := splitProviderPath(current.Path)
		_, _, nextProviderSpecific := splitProviderPath(next.Path)
		if currentProviderSpecific != nextProviderSpecific {
			return nextProviderSpecific, nil
		}
		return false, nil
	}
}

func validateTemplateType(tplType TemplateType) error {
	if _, ok := templateName[tplType]; !ok {
		return fmt.Errorf("invalid template type %d", tplType)
	}
	return nil
}

// TemplateResult represents the result of templating a single file
type TemplateResult struct {
	Path    string // Original relative path
	Content string // Templated content
	Error   error  // Any error that occurred during templating
}

type TemplatePathPredicate func(path string) bool

type TemplateOptions struct {
	Context        context.Context
	Type           TemplateType
	Provider       string
	CatalogOptions catalog.LoadOptions
	Data           any
	PathPredicate  TemplatePathPredicate
}

type templateSource struct {
	name     string
	fsys     fs.FS
	baseRoot string
}

func (tt TemplateType) String() string {
	return templateName[tt]
}

func loadTemplateSources(options TemplateOptions) ([]templateSource, error) {
	catalogSources, err := catalog.ResolveSources(options.CatalogOptions)
	if err != nil {
		return nil, err
	}
	sources := make([]templateSource, 0, len(catalogSources))

	for _, cat := range catalogSources {
		source, err := catalog.ResolveSource(cat)
		if err != nil {
			return nil, fmt.Errorf("resolve catalog source: %w", err)
		}
		sources = append(sources, templateSource{
			name:     cat,
			fsys:     os.DirFS(source.RootPath),
			baseRoot: ".",
		})
	}

	return sources, nil
}

func joinTemplateRoot(baseRoot string, elems ...string) string {
	if baseRoot == "." || baseRoot == "" {
		if len(elems) == 0 {
			return "."
		}
		return path.Join(elems...)
	}

	parts := append([]string{baseRoot}, elems...)
	return path.Join(parts...)
}

func templateRootsForType(baseRoot string, templateType TemplateType) []string {
	switch templateType {
	case All:
		return []string{
			joinTemplateRoot(baseRoot, DefaultPlatformConfigsPath),
			joinTemplateRoot(baseRoot, DefaultPlatformComponentsPath),
		}
	default:
		return []string{
			joinTemplateRoot(baseRoot, DefaultPlatformConfigsPath, templateType.String()),
			joinTemplateRoot(baseRoot, DefaultPlatformComponentsPath, templateType.String()),
		}
	}
}

func normalizeProviderName(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func splitProviderPath(relPath string) (string, string, bool) {
	normalized := filepath.ToSlash(relPath)
	parts := strings.Split(normalized, "/")
	for idx := 0; idx+1 < len(parts); idx++ {
		// Only treat this segment as a provider selector when it appears
		// directly inside the Terraform template directory.
		if idx == 0 || parts[idx-1] != Terraform.String() {
			continue
		}

		provider := strings.ToLower(parts[idx])
		if provider == "" {
			continue
		}

		// Robustly check that we are strictly within the platform-configs root directory
		// before stripping the provider segment.
		if parts[0] == DefaultPlatformConfigsPath {
			stripped := append([]string{}, parts[:idx]...)
			stripped = append(stripped, parts[idx+1:]...)
			return strings.Join(stripped, "/"), provider, true
		}

		// Under platform-components, we do NOT strip the provider segment from the path,
		// but we still want to detect the provider name to filter out other providers' assets!
		if parts[0] == DefaultPlatformComponentsPath {
			return normalized, provider, true
		}
	}

	return normalized, "", false
}

// StripProviderPath removes a Terraform provider selector segment from a
// relative template path (e.g. ".../terraform/stackit/...") if present.
func StripProviderPath(relPath string) string {
	stripped, _, _ := splitProviderPath(relPath)
	return stripped
}
