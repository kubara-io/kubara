package catalog

import (
	"encoding/json"
	"fmt"
	"strings"

	goYaml "go.yaml.in/yaml/v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type Status string

const (
	StatusEnabled  Status = "enabled"
	StatusDisabled Status = "disabled"
)

type ServiceDefinition struct {
	APIVersion string      `yaml:"apiVersion"`
	Kind       string      `yaml:"kind"`
	Metadata   Metadata    `yaml:"metadata"`
	Spec       ServiceSpec `yaml:"spec"`
}

type Metadata struct {
	Name        string            `yaml:"name"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

type ServiceSpec struct {
	ChartPath    string                           `yaml:"chartPath"`
	AppName      string                           `yaml:"appName,omitempty"`
	Default      Status                           `yaml:"default,omitempty"`
	Status       Status                           `yaml:"status,omitempty"` // compatibility with early proposal drafts
	ClusterTypes []string                         `yaml:"clusterTypes,omitempty"`
	ConfigSchema *apiextensionsv1.JSONSchemaProps `yaml:"configSchema,omitempty"`
}

func (s *ServiceSpec) UnmarshalYAML(value *goYaml.Node) error {
	type serviceSpecAlias struct {
		ChartPath    string         `yaml:"chartPath"`
		AppName      string         `yaml:"appName,omitempty"`
		Default      Status         `yaml:"default,omitempty"`
		Status       Status         `yaml:"status,omitempty"`
		ClusterTypes []string       `yaml:"clusterTypes,omitempty"`
		ConfigSchema map[string]any `yaml:"configSchema,omitempty"`
	}

	var raw serviceSpecAlias
	if err := value.Decode(&raw); err != nil {
		return err
	}

	s.ChartPath = raw.ChartPath
	s.AppName = raw.AppName
	s.Default = raw.Default
	s.Status = raw.Status
	s.ClusterTypes = raw.ClusterTypes

	if len(raw.ConfigSchema) == 0 {
		s.ConfigSchema = nil
		return nil
	}

	configSchemaBytes, err := json.Marshal(raw.ConfigSchema)
	if err != nil {
		return fmt.Errorf("marshal configSchema: %w", err)
	}

	var schema apiextensionsv1.JSONSchemaProps
	if err := json.Unmarshal(configSchemaBytes, &schema); err != nil {
		return fmt.Errorf("unmarshal configSchema into JSONSchemaProps: %w", err)
	}

	s.ConfigSchema = &schema
	return nil
}

func (s ServiceSpec) EffectiveDefaultStatus() Status {
	switch {
	case s.Default == StatusEnabled || s.Default == StatusDisabled:
		return s.Default
	case s.Status == StatusEnabled || s.Status == StatusDisabled:
		return s.Status
	default:
		return StatusDisabled
	}
}

type Catalog struct {
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
	if strings.TrimSpace(d.APIVersion) == "" {
		return fmt.Errorf("missing apiVersion")
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
	return nil
}

func ToMap(schema *apiextensionsv1.JSONSchemaProps) (map[string]any, error) {
	if schema == nil {
		return nil, nil
	}
	b, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}

	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}

	return out, nil
}
