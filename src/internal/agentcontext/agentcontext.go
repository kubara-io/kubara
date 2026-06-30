// Package agentcontext renders and writes the coding-agent onboarding file
// (AGENTS.md) into a kubara GitOps repository.
//
// The file gives AI coding agents (Claude Code, Codex, …) a compact,
// version-pinned entry point into kubara: it delegates command and config
// details to the self-describing CLI (`kubara --help`, `kubara schema`) and links
// the raw Markdown documentation sources on GitHub for the installed binary
// version, which is far cheaper for an agent to read than rendered HTML.
package agentcontext

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/kubara-io/kubara/internal/utils"
)

const (
	docsSiteURL = "https://docs.kubara.io"

	// AgentsFileName is the cross-agent onboarding file (read automatically by
	// Claude Code, Codex and similar tools).
	AgentsFileName = "AGENTS.md"
)

//go:embed templates/AGENTS.md.tmpl
var agentsTemplate string

// releaseTagPattern matches a clean semver release tag (optionally prefixed with
// "v"). Pre-release and snapshot versions deliberately do not match, so that
// documentation links fall back to a docs version that is guaranteed to resolve.
var releaseTagPattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)

// DocsRef resolves a clean release tag (e.g. "v0.10.0", normalised with a "v"
// prefix) or "main" for dev/snapshot/empty builds. Release builds inject the git
// tag via ldflags. It is the basis for DocsVersion.
func DocsRef(version string) string {
	v := strings.TrimSpace(version)
	if !releaseTagPattern.MatchString(v) {
		return "main"
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// DocsVersion resolves the hosted (mike) docs version segment used to link the
// published llms.txt index. Release builds map to the git tag (vX.Y.Z, the
// directory mike deploys under); everything else maps to the "latest-dev"
// alias, since "main" is a git ref but not a published docs version.
func DocsVersion(version string) string {
	if ref := DocsRef(version); ref != "main" {
		return ref
	}
	return "latest-dev"
}

type templateData struct {
	Version  string
	DocsSite string
	DocsBase string // docs site root for this version, e.g. https://docs.kubara.io/v0.10.0
	LlmsTxt  string
}

func newTemplateData(version string) templateData {
	display := strings.TrimSpace(version)
	if display == "" {
		display = "dev"
	}
	docsBase := fmt.Sprintf("%s/%s", docsSiteURL, DocsVersion(version))
	return templateData{
		Version:  display,
		DocsSite: docsSiteURL,
		DocsBase: docsBase,
		LlmsTxt:  docsBase + "/llms.txt",
	}
}

// Render returns the rendered AGENTS.md content for the given binary version.
func Render(version string) ([]byte, error) {
	tmpl, err := template.New(AgentsFileName).Option("missingkey=error").Parse(agentsTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, newTemplateData(version)); err != nil {
		return nil, fmt.Errorf("render template: %w", err)
	}
	return buf.Bytes(), nil
}

// WriteResult reports the outcome of writing the agent-context file.
type WriteResult struct {
	Path    string // absolute path of the file
	Written bool   // false when the file already existed and overwrite was false
}

// Write renders AGENTS.md and writes it into dir. An existing file is left
// untouched unless overwrite is true.
func Write(dir, version string, overwrite bool) (WriteResult, error) {
	content, err := Render(version)
	if err != nil {
		return WriteResult{}, err
	}

	path := filepath.Join(dir, AgentsFileName)
	result := WriteResult{Path: path}

	exists, err := utils.FileExist(path)
	if err != nil {
		return result, err
	}
	if exists && !overwrite {
		return result, nil
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return result, fmt.Errorf("write %s: %w", path, err)
	}
	result.Written = true
	return result, nil
}
