package catalog

import (
	"fmt"
	"maps"
	"strings"

	"kubara/internal/assets/service"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// ServiceDefinitionAPIVersion is the supported ServiceDefinition apiVersion.
const ServiceDefinitionAPIVersion = "kubara.io/v1alpha1"

// ServiceDefinition describes a catalog service entry.
type ServiceDefinition struct {
	// APIVersion declares the schema version.
	APIVersion string `json:"apiVersion"`
	// Kind is expected to be ServiceDefinition.
	Kind string `json:"kind"`
	// Metadata contains identity and optional annotations.
	Metadata Metadata `json:"metadata"`
	// Spec contains runtime-relevant service settings.
	Spec ServiceSpec `json:"spec"`
}

// Metadata contains metadata fields for a service definition.
type Metadata struct {
	// Name is the canonical service name.
	Name string `json:"name"`
	// Annotations carries optional metadata.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ServiceSpec contains the desired behavior and schema of a service.
type ServiceSpec struct {
	// ChartPath points to the Helm chart path under managed catalog.
	ChartPath string `json:"chartPath"`
	// AppName optionally overrides the default Argo CD application name.
	AppName string `json:"appName,omitempty"`
	// Status defines the default status for the service.
	Status service.Status `json:"status"`
	// ClusterTypes limits the service to specific cluster types.
	ClusterTypes []string `json:"clusterTypes,omitempty"`
	// ConfigSchema describes config values using OpenAPI v3 schema props.
	ConfigSchema *apiextensionsv1.JSONSchemaProps `json:"configSchema,omitempty"`
}

// Catalog represents a set of service definitions keyed by canonical service name.
type Catalog struct {
	// Services maps canonical service names to definitions.
	Services map[string]ServiceDefinition
}

func (c Catalog) Clone() Catalog {
	out := Catalog{Services: make(map[string]ServiceDefinition, len(c.Services))}
	maps.Copy(out.Services, c.Services)
	return out
}

func (d ServiceDefinition) Validate() error {
	apiVersion := strings.TrimSpace(d.APIVersion)
	if apiVersion == "" {
		return fmt.Errorf("missing apiVersion")
	}
	if apiVersion != ServiceDefinitionAPIVersion {
		return fmt.Errorf("apiVersion must be %q", ServiceDefinitionAPIVersion)
	}
	if strings.TrimSpace(d.Kind) != "ServiceDefinition" {
		return fmt.Errorf("kind must be ServiceDefinition")
	}
	if strings.TrimSpace(d.Metadata.Name) == "" {
		return fmt.Errorf("missing metadata.name")
	}
	if strings.TrimSpace(d.Spec.ChartPath) == "" {
		return fmt.Errorf("missing spec.chartPath")
	}
	if d.Spec.Status != service.StatusEnabled && d.Spec.Status != service.StatusDisabled {
		return fmt.Errorf(`spec.status must be either %q or %q`, service.StatusEnabled, service.StatusDisabled)
	}
	return nil
}
