package templates

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"go.yaml.in/yaml/v3"
)

//go:embed all:embedded
var templatesFSNew embed.FS

type TemplateType int

const (
	Terraform TemplateType = iota
	Helm
	All
)

const (
	tmplRoot                  string = "embedded"
	DefaultManagedCatalogPath string = "managed-service-catalog"
	DefaultOverlayValuesPath  string = "customer-service-catalog"
	providerFolderName        string = "providers"
)

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

func templateFuncMap() template.FuncMap {
	funcs := sprig.FuncMap()
	funcs["toYaml"] = func(v any) (string, error) {
		out, err := yaml.Marshal(v)
		if err != nil {
			return "", err
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

type selectedTemplate struct {
	sourcePath       string
	providerSpecific bool
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

		*out = append(*out, filepath.ToSlash(rel))
		return nil
	}
}

func GetEmbeddedTemplatesList(tplType TemplateType) ([]string, error) {
	var out []string
	var err error
	walkDirFunc := makeWalkDirFunc(tmplRoot, &out)
	embeddedCS := tmplRoot + "/" + DefaultOverlayValuesPath + "/" + tplType.String()
	embeddedMS := tmplRoot + "/" + DefaultManagedCatalogPath + "/" + tplType.String()
	switch tplType {
	case All:
		err = fs.WalkDir(templatesFSNew, tmplRoot, walkDirFunc)
	default:
		errWalkCS := fs.WalkDir(templatesFSNew, embeddedCS, walkDirFunc)
		errWalkMS := fs.WalkDir(templatesFSNew, embeddedMS, walkDirFunc)
		err = errors.Join(errWalkCS, errWalkMS)
	}

	return out, err
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

// GetEmbeddedTemplatesListForProvider returns template paths for one provider.
// Files under ".../providers/<provider>/..." are included only for that provider.
// Provider-specific files override common files with the same stripped path.
func GetEmbeddedTemplatesListForProvider(tplType TemplateType, provider string) ([]string, error) {
	files, err := GetEmbeddedTemplatesList(tplType)
	if err != nil {
		return nil, err
	}

	return selectTemplatesForProvider(files, provider), nil
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

// TemplateFiles processes all the specified template files using html/template
// fileList should be obtained from GetEmbeddedTemplatesList
// data contains the variables to be used in templating
func TemplateFiles(fileList []string, data any) ([]TemplateResult, error) {
	results := make([]TemplateResult, 0, len(fileList))
	var allErrors []error

	for _, relPath := range fileList {
		result := TemplateResult{Path: relPath}

		// Read the file content from embedded filesystem
		fullPath := filepath.Join(tmplRoot, relPath)
		content, err := fs.ReadFile(templatesFSNew, fullPath)
		if err != nil {
			result.Error = err
			results = append(results, result)
			allErrors = append(allErrors, err)
			continue
		}

		if strings.HasSuffix(fullPath, ".tplt") {
			// Parse the template
			// Using relPath as name to aid debugging
			//tmpl, err := template.New(relPath).Funcs(templateFuncMap()).Option("missingkey=error").Parse(string(content))
			tmpl, err := template.New(relPath).Funcs(templateFuncMap()).Parse(string(content))
			if err != nil {
				result.Error = err
				results = append(results, result)
				allErrors = append(allErrors, err)
				continue
			}

			// Execute the template with the provided data
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

// TemplateAllFiles is a convenience function that gets the file list and templates them
func TemplateAllFiles(tplType TemplateType, data any) ([]TemplateResult, error) {
	return TemplateAllFilesForProvider(tplType, data, "")
}

// TemplateAllFilesForProvider is a convenience function that gets the
// provider-filtered file list and templates it.
func TemplateAllFilesForProvider(tplType TemplateType, data any, provider string) ([]TemplateResult, error) {
	fileList, err := GetEmbeddedTemplatesListForProvider(tplType, provider)
	if err != nil {
		return nil, err
	}

	return TemplateFiles(fileList, data)
}
