package catalog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	catalogTypes "github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/service"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"go.yaml.in/yaml/v4"
)

func NewCatalogService() *cli.Command {
	cmd := &cli.Command{
		Name:        "add",
		Usage:       "Add a service definition to the current catalog",
		UsageText:   "kubara catalog add SERVICE_NAME",
		Description: "Creates services/SERVICE_NAME.yaml in the current catalog. Run this command from a catalog root that already contains Catalog.yaml.",
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name: "service-name",
				Config: cli.StringConfig{
					TrimSpace: true,
				},
			},
		},
		Action: func(c context.Context, cmd *cli.Command) error {
			serviceName := cmd.StringArg("service-name")
			if len(serviceName) == 0 {
				cli.ShowSubcommandHelpAndExit(cmd, 1)
			}

			return createService(serviceName)
		},
	}

	return cmd
}

func createService(serviceName string) error {
	if !catalogTypes.RFC1123Label.MatchString(serviceName) {
		return fmt.Errorf("service name must adhere to rfc 1123: must be 1-63 characters, start with a lowercase letter, contain only lowercase letters, digits, or '-', and end with a letter or digit")
	}

	if _, err := os.Stat("Catalog.yaml"); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("this directory is missing a Catalog.yaml")
	}

	servicePath := path.Join("services", fmt.Sprintf("%s.yaml", serviceName))
	if _, err := os.Stat(servicePath); err == nil {
		return fmt.Errorf("a service with name %s already exists", serviceName)
	}

	service := catalogTypes.ServiceDefinition{
		APIVersion: "kubara.io/v1alpha1",
		Kind:       "ServiceDefinition",
		Metadata: catalogTypes.Metadata{
			Name: serviceName,
		},
		Spec: catalogTypes.ServiceSpec{
			ChartPath: serviceName,
			Status:    service.StatusDisabled,
			ClusterTypes: []string{
				"hub",
				"spoke",
			},
		},
	}

	serviceRaw, err := yaml.Marshal(service)
	if err != nil {
		return fmt.Errorf("cannot marshal service: %w", err)
	}

	if err := os.WriteFile(servicePath, serviceRaw, 0o600); err != nil {
		return fmt.Errorf("cannot create Catalog.yaml: %w", err)
	}

	log.Info().Msg("Service has been added")

	return nil
}
