package catalog

import (
	"fmt"

	"github.com/kubara-io/kubara/internal/utils"
	"github.com/urfave/cli/v3"
)

func resolveCatalogCommandWorkingDir(cmd *cli.Command) (string, error) {
	ws, err := utils.ResolveWorkspaceFromCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}
	return ws.WorkDir, nil
}
