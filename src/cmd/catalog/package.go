package catalog

import (
	"context"
	"errors"
	"fmt"

	internal "github.com/kubara-io/kubara/internal/catalog"
	"github.com/rs/zerolog/log"

	"github.com/urfave/cli/v3"
)

func NewCatalogPackage() *cli.Command {
	return &cli.Command{
		Name:        "package",
		Aliases:     []string{"pkg"},
		Usage:       "Package the current catalog directory into the local cache",
		UsageText:   "kubara catalog package [oci://REGISTRY/BASE/PATH/]",
		Description: "Packages the current catalog directory into the local cache using the catalog version from Catalog.yaml and derives the final OCI reference from the optional base path. If omitted, kubara uses `oci://localhost/`.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() > 1 {
				cli.ShowSubcommandHelpAndExit(cmd, 1)
			}
			cwd, err := resolveCatalogCommandWorkingDir(cmd)
			if err != nil {
				return err
			}
			result, err := internal.PackageCatalog(internal.PackageOptions{
				CatalogRoot:   cwd,
				ReferenceBase: cmd.Args().First(),
			})
			if err != nil {
				if errors.Is(err, internal.ErrCatalogManifestNotFound) {
					return fmt.Errorf("%w; run this command from the catalog root or pass --work-dir /path/to/catalog", err)
				}
				return err
			}

			log.Info().Msgf(
				"Catalog %q version %q has been packaged into the local cache as %s with digest %s",
				result.Manifest.Metadata.Name,
				result.Manifest.Spec.Version,
				result.Reference,
				result.Artifact.ManifestDigest,
			)
			return nil
		},
	}
}
