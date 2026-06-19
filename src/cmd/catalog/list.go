package catalog

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	internal "github.com/kubara-io/kubara/internal/catalog"
	"github.com/rs/zerolog/log"

	"github.com/urfave/cli/v3"
)

func NewCatalogList() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Aliases:     []string{"ls"},
		Usage:       "List cached local and OCI-backed catalogs",
		UsageText:   "kubara catalog list",
		Description: "Lists cached catalogs from the local OCI cache, including local packages and cached OCI references.",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			entries, err := internal.ListCachedCatalogs()
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				log.Info().Msg("No cached catalogs found")
				return nil
			}

			writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(writer, "NAME\tVERSION\tREFERENCE\tDIGEST")
			for _, entry := range entries {
				_, _ = fmt.Fprintf(
					writer,
					"%s\t%s\t%s\t%s\n",
					entry.CatalogName,
					entry.CatalogVersion,
					entry.Reference,
					entry.ManifestDigest,
				)
			}
			return writer.Flush()
		},
	}
}
