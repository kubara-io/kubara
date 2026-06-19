package catalog

import (
	"context"
	"fmt"

	internal "github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/utils"
	"github.com/rs/zerolog/log"

	"github.com/urfave/cli/v3"
)

func NewCatalogUnpackage() *cli.Command {
	return &cli.Command{
		Name:        "unpackage",
		Aliases:     []string{"unpkg"},
		Usage:       "Materialize a cached OCI catalog as an editable directory",
		UsageText:   "kubara catalog unpackage oci://registry/repository:x.y.z [directory]",
		Description: "Copies a cached OCI catalog into an editable directory.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() < 1 || cmd.Args().Len() > 2 {
				cli.ShowSubcommandHelpAndExit(cmd, 1)
			}

			cwd, err := resolveCatalogCommandWorkingDir(cmd)
			if err != nil {
				return err
			}

			outputPath := ""
			if cmd.Args().Len() == 2 {
				outputPath, err = utils.GetFullPath(cmd.Args().Get(1), cwd)
				if err != nil {
					return fmt.Errorf("get output directory: %w", err)
				}
			}

			result, err := internal.UnpackageCatalog(internal.UnpackageOptions{
				Reference:  cmd.Args().First(),
				OutputPath: outputPath,
				WorkDir:    cwd,
			})
			if err != nil {
				return err
			}

			log.Info().Msgf(
				"Catalog %q version %q has been unpackaged to %s",
				result.Artifact.CatalogName,
				result.Artifact.CatalogVersion,
				result.OutputPath,
			)
			return nil
		},
	}
}
