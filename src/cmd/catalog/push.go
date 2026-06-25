package catalog

import (
	"context"

	internal "github.com/kubara-io/kubara/internal/catalog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

type catalogPushFlags struct {
	from     string
	insecure bool
}

func NewCatalogPush() *cli.Command {
	flags := &catalogPushFlags{}
	cmd := &cli.Command{
		Name:        "push",
		Usage:       "Push catalog to a remote registry",
		UsageText:   "kubara catalog push [--from oci://registry-source/repository:x.y.z] [--insecure] oci://registry-target/repository:x.y.z",
		Description: "Pushes a cached/packaged catalog artifact to a remote OCI compliant registry.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "from",
				Usage:       "Push an existing cached catalog reference to another registry.",
				Destination: &flags.from,
			},
			&cli.BoolFlag{
				Name:        "insecure",
				Usage:       "Ignore TLS certificate verification issues for registry connections.",
				Destination: &flags.insecure,
			},
		},
		Action: func(c context.Context, cmd *cli.Command) error {
			if cmd.Args().Len() != 1 {
				cli.ShowSubcommandHelpAndExit(cmd, 1)
			}

			result, err := internal.PushCatalog(c, internal.PushOptions{
				Reference: cmd.Args().First(),
				From:      flags.from,
				Insecure:  flags.insecure,
			})
			if err != nil {
				return err
			}

			log.Info().Msgf(
				"Catalog %q version %q has been pushed to %s with digest %s",
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
