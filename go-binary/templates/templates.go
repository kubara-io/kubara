package templates

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
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
)

var templateName = map[TemplateType]string{
	Terraform: "terraform",
	Helm:      "helm",
	All:       "all",
}

// TemplateResult represents the result of templating a single file
type TemplateResult struct {
	Path    string // Original relative path
	Content string // Templated content
	Error   error  // Any error that occurred during templating
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
			//tmpl, err := template.New(relPath).Funcs(sprig.FuncMap()).Option("missingkey=error").Parse(string(content))
			tmpl, err := template.New(relPath).Funcs(sprig.FuncMap()).Parse(string(content))
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
	fileList, err := GetEmbeddedTemplatesList(tplType)
	if err != nil {
		return nil, err
	}

	return TemplateFiles(fileList, data)
}
