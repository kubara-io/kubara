package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"sigs.k8s.io/yaml"

	cat "github.com/kubara-io/kubara/internal/catalog"
	svc "github.com/kubara-io/kubara/internal/service"
)

func CreateService(serviceName string) error {
	if !cat.RFC1123Label.MatchString(serviceName) {
		return fmt.Errorf("service name must adhere to rfc 1123: must be 1-63 characters, start with a lowercase letter, contain only lowercase letters, digits, or '-', and end with a letter or digit")
	}

	if _, err := os.Stat("Catalog.yaml"); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("this directory is missing a Catalog.yaml")
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
