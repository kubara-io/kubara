package migrations

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// ConfigVersion constants
const (
	ConfigVersionV1Alpha1 = "v1alpha1"
	ConfigVersionV1Alpha2 = "v1alpha2"
	ConfigVersionV1Alpha3 = "v1alpha3"
	ConfigVersionV1Alpha4 = "v1alpha4"
	ConfigVersionV1Alpha5 = "v1alpha5"
)

// Apply runs all registered schema and repository layout migrations.
func Apply(cwd string, config map[string]any) (bool, error) {
	var migrated bool
	if isLegacyConfig(config) {
		if err := migrateLegacyConfig(config); err != nil {
			return false, fmt.Errorf("migrate legacy config: %w", err)
		}
		migrated = true
	}

	if isV1Alpha1Config(config) {
		if err := migrateV1Alpha1Config(config); err != nil {
			return false, fmt.Errorf("migrate V1Alpha1 config: %w", err)
		}
		migrated = true
	}

	if isV1Alpha2Config(config) {
		if err := migrateV1Alpha2Config(cwd, config); err != nil {
			return false, fmt.Errorf("migrate V1Alpha2 config: %w", err)
		}
		migrated = true
	}

	if isV1Alpha3Config(config) {
		if err := migrateV1Alpha3Config(config); err != nil {
			return false, fmt.Errorf("migrate V1Alpha3 config: %w", err)
		}
		migrated = true
	}

	if isV1Alpha4Config(config) {
		if err := migrateV1Alpha4Config(config); err != nil {
			return false, fmt.Errorf("migrate V1Alpha4 config: %w", err)
		}
		migrated = true
	}

	return migrated, nil
}

func isLegacyConfig(raw map[string]any) bool {
	_, hasVersion := raw["version"]
	return !hasVersion
}

func isV1Alpha1Config(raw map[string]any) bool {
	version, hasVersion := raw["version"]
	return version == ConfigVersionV1Alpha1 && hasVersion
}

func isV1Alpha2Config(raw map[string]any) bool {
	version, hasVersion := raw["version"]
	return version == ConfigVersionV1Alpha2 && hasVersion
}

func isV1Alpha3Config(raw map[string]any) bool {
	version, hasVersion := raw["version"]
	return version == ConfigVersionV1Alpha3 && hasVersion
}

func isV1Alpha4Config(raw map[string]any) bool {
	version, hasVersion := raw["version"]
	return version == ConfigVersionV1Alpha4 && hasVersion
}

func clusterLabel(cluster map[string]any, clusterIndex int) string {
	if name, ok := cluster["name"].(string); ok && strings.TrimSpace(name) != "" {
		return fmt.Sprintf("cluster %q", name)
	}

	return fmt.Sprintf("clusters[%d]", clusterIndex)
}

func removeDirIfEmpty(dir string) error {
	if err := os.Remove(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTEMPTY) {
			return nil
		}

		return fmt.Errorf("remove empty dir %q: %w", dir, err)
	}

	return nil
}

func moveDirContents(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("readdir %q: %w", srcDir, err)
	}

	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %q: %w", dstDir, err)
	}

	for _, entry := range entries {
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())

		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("destination already exists: %q", dst)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat %q: %w", dst, err)
		}

		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("move %q -> %q: %w", src, dst, err)
		}
	}

	return nil
}
