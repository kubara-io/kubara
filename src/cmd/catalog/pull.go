package catalog

import (
	"context"

	internal "github.com/kubara-io/kubara/internal/catalog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

type catalogPullFlags struct {
	insecure bool
}

func NewCatalogPull() *cli.Command {
	flags := &catalogPullFlags{}
	cmd := &cli.Command{
		Name:        "pull",
		Usage:       "Pull a catalog from a remote registry",
		UsageText:   "kubara catalog pull [--insecure] oci://registry/repository:x.y.z",
		Description: "Pulls a catalog artifact into the local cache from any OCI compliant registry.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "insecure",
				Usage:       "Ignore TLS certificate verification issues for the registry connection.",
				Destination: &flags.insecure,
			},
		},
		Action: func(c context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				cli.ShowSubcommandHelpAndExit(cmd, 1)
			}

			result, err := internal.PullCatalog(c, internal.PullOptions{
				Reference: cmd.Args().First(),
				Insecure:  flags.insecure,
			})
			if err != nil {
				return err
			}

			message := "Catalog %q version %q has been pulled into the local cache as %s with digest %s"
			if result.Updated {
				message = "Catalog %q version %q has been updated from remote as %s with digest %s"
			}

			log.Info().Msgf(
				message,
				result.Artifact.CatalogName,
				result.Artifact.CatalogVersion,
				result.Reference,
				result.Artifact.ManifestDigest,
			)
			return nil
		},
	}

	return cmd
}
