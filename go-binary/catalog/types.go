package catalog

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Status string

const (
	// StatusEnabled marks a service as enabled.
	StatusEnabled Status = "enabled"
	// StatusDisabled marks a service as disabled.
	StatusDisabled Status = "disabled"
)

// ServiceDefinitionAPIVersion is the supported ServiceDefinition apiVersion.
const ServiceDefinitionAPIVersion = "kubara.io/v1alpha1"

// ServiceDefinition describes a catalog service entry.
type ServiceDefinition struct {
	// APIVersion declares the schema version.
	APIVersion string `yaml:"apiVersion"`
	// Kind is expected to be ServiceDefinition.
	Kind string `yaml:"kind"`
	// Metadata contains identity and optional annotations.
	Metadata Metadata `yaml:"metadata"`
	// Spec contains runtime-relevant service settings.
	Spec ServiceSpec `yaml:"spec"`
}

// Metadata contains metadata fields for a service definition.
type Metadata struct {
	// Name is the canonical service name.
	Name string `yaml:"name"`
	// Annotations carries optional metadata.
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// ServiceSpec contains the desired behavior and schema of a service.
type ServiceSpec struct {
	// ChartPath points to the Helm chart path under managed catalog.
	ChartPath string `yaml:"chartPath"`
	// AppName optionally overrides the default Argo CD application name.
	AppName string `yaml:"appName,omitempty"`
	// Status defines the default status for the service.
	Status Status `yaml:"status"`
	// ClusterTypes limits the service to specific cluster types.
	ClusterTypes []string `yaml:"clusterTypes,omitempty"`
	// ConfigSchema describes config values using OpenAPI v3 schema props.
	ConfigSchema *apiextensionsv1.JSONSchemaProps `yaml:"configSchema,omitempty"`
}

type serviceSpecYAML struct {
	ChartPath    string         `yaml:"chartPath"`
	AppName      string         `yaml:"appName,omitempty"`
	Status       Status         `yaml:"status"`
	ClusterTypes []string       `yaml:"clusterTypes,omitempty"`
	ConfigSchema map[string]any `yaml:"configSchema,omitempty"`
}

func (s *ServiceSpec) UnmarshalYAML(value *yaml.Node) error {
	var raw serviceSpecYAML
	if err := value.Decode(&raw); err != nil {
		return err
	}

	s.ChartPath = raw.ChartPath
	s.AppName = raw.AppName
	s.Status = raw.Status
	s.ClusterTypes = raw.ClusterTypes

	if len(raw.ConfigSchema) == 0 {
		s.ConfigSchema = nil
		return nil
	}

	var schema apiextensionsv1.JSONSchemaProps
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(raw.ConfigSchema, &schema); err != nil {
		return fmt.Errorf("decode configSchema into JSONSchemaProps: %w", err)
	}

	s.ConfigSchema = &schema
	return nil
}

// Catalog represents a set of service definitions keyed by canonical service name.
type Catalog struct {
	// Services maps canonical service names to definitions.
	Services map[string]ServiceDefinition
}

func (c Catalog) Clone() Catalog {
	out := Catalog{Services: make(map[string]ServiceDefinition, len(c.Services))}
	for name, def := range c.Services {
		out.Services[name] = def
	}
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
	if d.Spec.Status != StatusEnabled && d.Spec.Status != StatusDisabled {
		return fmt.Errorf(`spec.status must be either %q or %q`, StatusEnabled, StatusDisabled)
	}
	return nil
}

func ToMap(schema *apiextensionsv1.JSONSchemaProps) (map[string]any, error) {
	if schema == nil {
		return nil, nil
	}

	out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(schema)
	if err != nil {
		return nil, fmt.Errorf("convert schema to map: %w", err)
	}
	return out, nil
}
