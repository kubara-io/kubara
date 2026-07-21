package catalog

import (
	"fmt"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

func resolveCatalogCommandWorkingDir(cmd *cli.Command) (string, error) {
	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return cwd, nil
}
