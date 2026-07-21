package catalog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	cat "github.com/kubara-io/kubara/internal/catalog"
	svc "github.com/kubara-io/kubara/internal/service"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
	"sigs.k8s.io/yaml"
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

			return CreateService(serviceName)
		},
	}

	return cmd
}

func CreateService(serviceName string) error {
	if !cat.RFC1123Label.MatchString(serviceName) {
		return fmt.Errorf("service name must adhere to rfc 1123: must be 1-63 characters, start with a lowercase letter, contain only lowercase letters, digits, or '-', and end with a letter or digit")
	}

	if _, err := os.Stat("Catalog.yaml"); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("this directory is missing a Catalog.yaml")
	}
	if _, err := cat.LoadCatalogManifest("."); err != nil {
		return err
	}

	servicePath := filepath.Join("services", fmt.Sprintf("%s.yaml", serviceName))
	if _, err := os.Stat(servicePath); err == nil {
		return fmt.Errorf("a service with name %s already exists", serviceName)
	}

	service := cat.ServiceDefinition{
		APIVersion: cat.ServiceDefinitionAPIVersion,
		Kind:       cat.ServiceDefinitionKind,
		Metadata: cat.Metadata{
			Name: serviceName,
		},
		Spec: cat.ServiceSpec{
			ChartPath: serviceName,
			Status:    svc.StatusDisabled,
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

	if err := os.MkdirAll(filepath.Join("services"), 0o755); err != nil {
		return fmt.Errorf("cannot create services directory: %w", err)
	}

	if err := os.WriteFile(servicePath, serviceRaw, 0o600); err != nil {
		return fmt.Errorf("cannot create service: %w", err)
	}

	log.Info().Msgf("Service %q has been successfully added to the catalog", serviceName)

	return nil
}
