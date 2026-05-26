package render

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/kubara-io/kubara/internal/catalog"

	"github.com/Masterminds/sprig/v3"
	"go.yaml.in/yaml/v3"
)

type TemplateType int

const (
	Terraform TemplateType = iota
	Helm
	All
)

const (
	tmplRoot                  string = "built-in"
	DefaultManagedCatalogPath string = "managed-service-catalog"
	DefaultOverlayValuesPath  string = "customer-service-catalog"
	providerFolderName        string = "providers"
)

var templatesFSNew fs.FS = catalog.BuiltInFS()

// SupportedProviders lists every provider that has embedded templates.
// Add new entries here when introducing provider-specific template directories.
var SupportedProviders = map[string]bool{
	"stackit": true,
}

var templateName = map[TemplateType]string{
	Terraform: "terraform",
	Helm:      "helm",
	All:       "all",
}

func TemplateFiles(options TemplateOptions) ([]TemplateResult, error) {
	fileList, err := getTemplateFiles(options)
	if err != nil {
		return nil, fmt.Errorf("get template files for provider %q: %w", options.Provider, err)
	}

	selected, err := selectTemplateFilesForProvider(fileList, options.Provider, options.Overwrite)
	if err != nil {
		return nil, fmt.Errorf("select templates for provider %q: %w", options.Provider, err)
	}

	return templateResultsFromFiles(selected, options.Data)
}

func validateTemplateType(tplType TemplateType) error {
	if _, ok := templateName[tplType]; !ok {
		return fmt.Errorf("invalid template type %d", tplType)
	}
	return nil
}

func templateFuncMap() template.FuncMap {
	funcs := sprig.FuncMap()
	funcs["toYaml"] = func(v any) (string, error) {
		out, err := yaml.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("marshal value to YAML: %w", err)
		}
		return strings.TrimSuffix(string(out), "\n"), nil
	}
	return funcs
}

// TemplateResult represents the result of templating a single file
type TemplateResult struct {
	Path    string // Original relative path
	Content string // Templated content
	Error   error  // Any error that occurred during templating
}

type TemplateOptions struct {
	Type        TemplateType
	Provider    string
	CatalogPath string
	Overwrite   bool
	Data        any
}

type selectedTemplate struct {
	sourcePath       string
	providerSpecific bool
}

type templateSource struct {
	name     string
	fsys     fs.FS
	baseRoot string
	external bool
}

type templateFile struct {
	sourcePath string
	readPath   string
	fsys       fs.FS
	external   bool
}

func (tt TemplateType) String() string {
	return templateName[tt]
}

func makeWalkDirFunc(tmplRoot string, out *[]string) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(tmplRoot, path)
		if err != nil {
			return err
		}

		if strings.HasPrefix(rel, "services") {
			// TODO: Consider a cleaner separation of "template files" vs "catalog data"
			// and avoid relying on path-based filtering. For now, we can simply
			// ignore files under "services" as they are not templates but catalog data.
			return nil
		}

		*out = append(*out, filepath.ToSlash(rel))
		return nil
	}
}

func loadTemplateSources(options TemplateOptions) ([]templateSource, error) {
	sources := []templateSource{{
		name:     "built-in",
		fsys:     templatesFSNew,
		baseRoot: tmplRoot,
	}}

	if strings.TrimSpace(options.CatalogPath) == "" {
		return sources, nil
	}

	external := templateSource{
		name:     "external",
		fsys:     os.DirFS(options.CatalogPath),
		baseRoot: ".",
		external: true,
	}

	hasTemplates, err := sourceHasTemplateRoots(external)
	if err != nil {
		return nil, err
	}
	if !hasTemplates {
		return sources, nil
	}

	return append(sources, external), nil
}

func sourceHasTemplateRoots(source templateSource) (bool, error) {
	roots := []string{DefaultOverlayValuesPath, DefaultManagedCatalogPath}
	for _, root := range roots {
		info, err := fs.Stat(source.fsys, root)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return false, fmt.Errorf("stat %s template root %q: %w", source.name, root, err)
		}
		if !info.IsDir() {
			return false, fmt.Errorf("%s template root %q is not a directory", source.name, root)
		}
		return true, nil
	}

	return false, nil
}

func joinTemplateRoot(baseRoot string, elems ...string) string {
	if baseRoot == "." || baseRoot == "" {
		if len(elems) == 0 {
			return "."
		}
		return filepath.Join(elems...)
	}

	parts := append([]string{baseRoot}, elems...)
	return filepath.Join(parts...)
}

func makeTemplateFileWalkDirFunc(source templateSource, out *[]templateFile) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(source.baseRoot, path)
		if err != nil {
			return err
		}

		normalized := filepath.ToSlash(rel)
		if strings.HasPrefix(normalized, "services") {
			return nil
		}

		*out = append(*out, templateFile{
			sourcePath: normalized,
			readPath:   filepath.ToSlash(path),
			fsys:       source.fsys,
			external:   source.external,
		})
		return nil
	}
}

func getTemplateFiles(options TemplateOptions) ([]templateFile, error) {
	if err := validateTemplateType(options.Type); err != nil {
		return nil, err
	}

	sources, err := loadTemplateSources(options)
	if err != nil {
		return nil, fmt.Errorf("load template sources: %w", err)
	}

	out := make([]templateFile, 0)
	for _, source := range sources {
		walkDirFunc := makeTemplateFileWalkDirFunc(source, &out)
		var walkErr error

		switch options.Type {
		case All:
			walkErr = fs.WalkDir(source.fsys, source.baseRoot, walkDirFunc)
		default:
			roots := []string{
				joinTemplateRoot(source.baseRoot, DefaultOverlayValuesPath, options.Type.String()),
				joinTemplateRoot(source.baseRoot, DefaultManagedCatalogPath, options.Type.String()),
			}
			for _, root := range roots {
				if err := fs.WalkDir(source.fsys, root, walkDirFunc); err != nil {
					if source.external && errors.Is(err, fs.ErrNotExist) {
						continue
					}
					walkErr = errors.Join(walkErr, err)
				}
			}
		}

		if walkErr != nil {
			return nil, fmt.Errorf("walk %s templates for type %q: %w", source.name, options.Type.String(), walkErr)
		}
	}

	return out, nil
}

func normalizeProviderName(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func splitProviderPath(relPath string) (string, string, bool) {
	normalized := filepath.ToSlash(relPath)
	parts := strings.Split(normalized, "/")
	for idx := 0; idx+1 < len(parts); idx++ {
		if parts[idx] != providerFolderName {
			continue
		}

		// Only treat this segment as a provider selector when it appears
		// directly inside a template-type directory (terraform or helm).
		// This prevents stripping unrelated "providers/<x>" segments that
		// may appear elsewhere in the path.
		if idx == 0 || (parts[idx-1] != "terraform" && parts[idx-1] != "helm") {
			continue
		}

		provider := strings.ToLower(parts[idx+1])
		if provider == "" {
			return normalized, "", false
		}

		stripped := append([]string{}, parts[:idx]...)
		stripped = append(stripped, parts[idx+2:]...)
		return strings.Join(stripped, "/"), provider, true
	}

	return normalized, "", false
}

// StripProviderPath removes a provider selector segment from a relative
// template path (e.g. ".../providers/stackit/...") if present.
func StripProviderPath(relPath string) string {
	stripped, _, _ := splitProviderPath(relPath)
	return stripped
}

func selectTemplatesForProvider(files []string, provider string) []string {
	selectedProvider := normalizeProviderName(provider)
	sortedFiles := append([]string(nil), files...)
	sort.Strings(sortedFiles)

	selected := make(map[string]selectedTemplate, len(sortedFiles))
	keys := make([]string, 0, len(sortedFiles))

	for _, sourcePath := range sortedFiles {
		strippedPath, sourceProvider, isProviderSpecific := splitProviderPath(sourcePath)
		if isProviderSpecific && sourceProvider != selectedProvider {
			continue
		}

		current, exists := selected[strippedPath]
		if !exists {
			selected[strippedPath] = selectedTemplate{
				sourcePath:       sourcePath,
				providerSpecific: isProviderSpecific,
			}
			keys = append(keys, strippedPath)
			continue
		}

		// Provider-specific files override common files for the same output path.
		if isProviderSpecific && !current.providerSpecific {
			selected[strippedPath] = selectedTemplate{
				sourcePath:       sourcePath,
				providerSpecific: true,
			}
		}
	}

	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, selected[key].sourcePath)
	}

	return out
}

func shouldPreferTemplateFile(current templateFile, next templateFile, currentProviderSpecific bool, nextProviderSpecific bool, overwrite bool, strippedPath string) (bool, error) {
	if current.external != next.external {
		if !overwrite {
			return false, fmt.Errorf("template %q already exists in built-in catalog", strippedPath)
		}
		return next.external, nil
	}

	if currentProviderSpecific != nextProviderSpecific {
		return nextProviderSpecific, nil
	}

	return false, nil
}

func selectTemplateFilesForProvider(files []templateFile, provider string, overwrite bool) ([]templateFile, error) {
	selectedProvider := normalizeProviderName(provider)
	sortedFiles := append([]templateFile(nil), files...)
	sort.Slice(sortedFiles, func(i, j int) bool {
		if sortedFiles[i].sourcePath == sortedFiles[j].sourcePath {
			if sortedFiles[i].external == sortedFiles[j].external {
				return sortedFiles[i].readPath < sortedFiles[j].readPath
			}
			return !sortedFiles[i].external && sortedFiles[j].external
		}
		return sortedFiles[i].sourcePath < sortedFiles[j].sourcePath
	})

	selected := make(map[string]templateFile, len(sortedFiles))
	keys := make([]string, 0, len(sortedFiles))

	for _, file := range sortedFiles {
		strippedPath, sourceProvider, isProviderSpecific := splitProviderPath(file.sourcePath)
		if isProviderSpecific && sourceProvider != selectedProvider {
			continue
		}

		current, exists := selected[strippedPath]
		if !exists {
			selected[strippedPath] = file
			keys = append(keys, strippedPath)
			continue
		}

		_, _, currentProviderSpecific := splitProviderPath(current.sourcePath)
		replaceCurrent, err := shouldPreferTemplateFile(current, file, currentProviderSpecific, isProviderSpecific, overwrite, strippedPath)
		if err != nil {
			return nil, err
		}
		if replaceCurrent {
			selected[strippedPath] = file
		}
	}

	sort.Strings(keys)
	out := make([]templateFile, 0, len(keys))
	for _, key := range keys {
		out = append(out, selected[key])
	}

	return out, nil
}

func templateResultsFromFiles(fileList []templateFile, data any) ([]TemplateResult, error) {
	results := make([]TemplateResult, 0, len(fileList))
	var allErrors []error

	for _, file := range fileList {
		result := TemplateResult{Path: file.sourcePath}

		content, err := fs.ReadFile(file.fsys, file.readPath)
		if err != nil {
			result.Error = err
			results = append(results, result)
			allErrors = append(allErrors, err)
			continue
		}

		if strings.HasSuffix(file.readPath, ".tplt") {
			tmpl, err := template.New(file.sourcePath).Funcs(templateFuncMap()).Parse(string(content))
			if err != nil {
				result.Error = err
				results = append(results, result)
				allErrors = append(allErrors, err)
				continue
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				result.Error = err
				results = append(results, result)
				allErrors = append(allErrors, err)
				continue
			}

			result.Content = buf.String()
			results = append(results, result)
		} else {
			result.Content = string(content)
			results = append(results, result)
		}
	}

	var combinedError error
	if len(allErrors) > 0 {
		combinedError = errors.Join(allErrors...)
	}

	return results, combinedError
}
