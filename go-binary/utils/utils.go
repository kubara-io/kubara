package utils

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/rs/zerolog/log"
)

// DecodeB64 decodes a b64 string into raw string
func DecodeB64(from string) (string, error) {
	dec, err := base64.StdEncoding.DecodeString(from)
	if err != nil {
		return "", fmt.Errorf("decoding string %s failed: %w", from, err)
	}
	return string(dec), nil
}

// FileExist returns true if the the File at path exist.
// The absolute path of path is used for finding the File.
// If path is a directory false is returned, no error.
// If any error occurs, the result will be false followed by the first error
func FileExist(path string) (bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}

	file, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if file.IsDir() {
		return false, nil
	}
	return true, nil
}

// GetFullPath returns the absolute path representation of "path"
// if path is a relative path it returns the full path of "path" relative to "workDir"
func GetFullPath(path, workDir string) (string, error) {
	// Expand ~ to home directory (Unix/Linux/macOS convention)
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		path = filepath.Join(home, path[2:])
	}

	if filepath.IsAbs(path) {
		return path, nil
	}

	return filepath.Abs(filepath.Join(workDir, path))
}

func IsZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int:
		return v.Int() == 0
	case reflect.Bool:
		return !v.Bool()
	default:
		return false
	}
}

func IsDefaultValue(v reflect.Value, defaultVal any) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == defaultVal
	case reflect.Int:
		return v.Int() == defaultVal
	case reflect.Bool:
		return v.Bool() == defaultVal
	default:
		return false
	}
}

func SetFieldValue(field reflect.Value, value string) {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int:
		if intVal, err := strconv.Atoi(value); err == nil {
			field.SetInt(int64(intVal))
		}
	case reflect.Bool:
		if boolVal, err := strconv.ParseBool(value); err == nil {
			field.SetBool(boolVal)
		}
	}
}

func AddGitignore(cwd string) error {

	filename, _ := GetFullPath(".gitignore", cwd)

	if _, err := os.Stat(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			errW := os.WriteFile(filename, []byte(gitignoreKubara), 0600)
			if errW != nil {
				return errW
			}
		} else {
			return err
		}
	}

	// Create tmp gitignore
	temp, errTemp := os.CreateTemp(os.TempDir(), "tmp")
	if errTemp != nil {
		return errTemp
	}
	defer func() {
		if err := os.Remove(temp.Name()); err != nil {
			log.Warn().Err(err).Str("file", temp.Name()).Msg("failed to remove temporary file")
		}
	}()

	if _, err := temp.Write([]byte(gitignoreKubara)); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}

	// Merge the files

	err := mergeGitignoreFiles([]string{filename, temp.Name()}, ".gitignore")
	if err != nil {
		return err
	}
	log.Info().Str("file", filename).Msg("✓ copied prep file")
	return nil

}

func mergeGitignoreFiles(filePaths []string, outputPath string) error {
	var allLines []string
	seenLines := make(map[string]bool)

	for _, filePath := range filePaths {
		lines, err := readGitignoreLines(filePath)
		if err != nil {
			return fmt.Errorf("error reading %s: %w", filePath, err)
		}

		// Add unique lines
		for _, line := range lines {
			if !seenLines[line] {
				seenLines[line] = true
				allLines = append(allLines, line)
			}
		}
	}

	return writeGitignoreFile(outputPath, allLines)
}

func readGitignoreLines(filePath string) ([]string, error) {
	// Validate filePath is not attempting path traversal
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Warn().Err(closeErr).Str("file", absPath).Msg("failed to close file")
		}
	}()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Keep empty lines but avoid duplicates
		if trimmed == "" {
			// Only add empty line if previous line wasn't empty
			if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
				lines = append(lines, "")
			}
			continue
		}

		// Validate non-comment lines
		if !strings.HasPrefix(trimmed, "#") {
			pattern := gitignore.ParsePattern(line, nil)
			if pattern == nil {
				fmt.Printf("Warning: Invalid pattern ignored: %s\n", line)
				continue
			}
		}

		lines = append(lines, line)
	}

	return lines, scanner.Err()
}

func writeGitignoreFile(outputPath string, lines []string) error {
	// Validate outputPath is not attempting path traversal
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	// Ensure we're only writing to .gitignore in current directory
	if filepath.Base(absPath) != ".gitignore" {
		return fmt.Errorf("invalid output filename: must be .gitignore")
	}

	// Get current working directory and resolve symlinks
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	resolvedCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return fmt.Errorf("failed to resolve working directory: %w", err)
	}

	// Resolve symlinks in the target path
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	// Ensure the output path is within the current working directory
	if !strings.HasPrefix(resolvedPath+string(filepath.Separator), resolvedCwd+string(filepath.Separator)) {
		return fmt.Errorf("output path outside working directory")
	}

	file, err := os.Create(resolvedPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Warn().Err(closeErr).Str("file", resolvedPath).Msg("failed to close file")
		}
	}()

	writer := bufio.NewWriter(file)

	// Write lines directly
	for _, line := range lines {
		if _, err := fmt.Fprintln(writer, line); err != nil {
			return fmt.Errorf("failed to write line to file: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	return nil
}

const gitignoreKubara = `
#########################################
### Kubara gitignore ###
# kubara program
kubara_*amd64*
kubara_*arm64*
kubara

# Terraform

# Local .terraform directories
**/.terraform/

# State files
*.tfstate
*.tfstate.*

# Env helper scripts (optional, but often local only)
**/set-env*.sh
**/set-env*.ps1

# Terraform lock file (only ignore if using providers globally)
.terraform.lock.hcl

# Crash logs
crash.log
crash.*.log

# Terraform-generated files
generated_backend.tf
*.hprof
runtime_settings.properties

#########################################
# Helm
#########################################

# Helm charts and lock files
**/charts/
**/Chart.lock
**/*.tgz

#########################################
# Secrets
#########################################

# Generic
**/secrets
/secrets

# Specific
launchers/demo-e2e/edc-config.properties
.env

#########################################
# Logs
#########################################

*.log
test/

#########################################
# OS & Editor Artifacts
#########################################

# macOS
.DS_Store

# Visual Studio
.vs/
.vscode/

# IntelliJ
.idea/
*.iml
*.ipr
*.iws
*/out/
out/

#########################################
# Archives / Packaging
#########################################

*.jar
*.war
*.nar
*.ear
*.zip
*.tar.gz
*.rar

#########################################
# Go (if you use Go modules or compiled binaries)
#########################################

# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test

# Go build output
/go.sum
/go.mod
bin/
`
