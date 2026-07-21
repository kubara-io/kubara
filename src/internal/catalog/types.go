package catalog

import (
	"fmt"
	"maps"
	"regexp"
	"strings"

	"github.com/kubara-io/kubara/internal/service"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Adheres to RFC 1123 and kubernetes conventions
// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
var RFC1123Label = regexp.MustCompile(
	`^[a-z](?:[a-z0-9-]{0,61}[a-z0-9])?$`,
)

var StrictCatalogVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// CatalogArtifactElement
type CatalogArtifactElement string

const (
	CatalogDefinition           CatalogArtifactElement = "Catalog.yaml"
	ServicesDirectory           CatalogArtifactElement = "services"
	PlatformComponentsDirectory CatalogArtifactElement = "platform-components"
	PlatformConfigsDirectory    CatalogArtifactElement = "platform-configs"
)

// Support APIVersion for CatalogManifest
const CatalogAPIVersion = "kubara.io/v1alpha1"
const CatalogKind = "Catalog"

// Support APIVersion for ServiceDefinition
const ServiceDefinitionAPIVersion = "kubara.io/v1alpha1"
const ServiceDefinitionKind = "ServiceDefinition"

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
	// ChartPath points to the immutable Helm chart path under platform-components/helm/.
	ChartPath string `json:"chartPath"`
	// Status defines the default status for the service and may be overridden per cluster.
	Status service.Status `json:"status"`
	// ClusterTypes limits the service to specific cluster types as catalog metadata.
	ClusterTypes []string `json:"clusterTypes,omitempty"`
	// ConfigSchema describes config values using OpenAPI v3 schema props.
	ConfigSchema *apiextensionsv1.JSONSchemaProps `json:"configSchema,omitempty"`
}

// CatalogManifest describes a catalog root manifest.
type CatalogManifest struct {
	APIVersion string      `json:"apiVersion"`
	Kind       string      `json:"kind"`
	Metadata   Metadata    `json:"metadata"`
	Spec       CatalogSpec `json:"spec"`
}

// CatalogSpec contains catalog-level settings.
type CatalogSpec struct {
	Version string `json:"version"`
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

func validateApiObject(apiVersion, kind, name, expectedAPIVersion, expectedKind string) error {
	apiVersion = strings.TrimSpace(apiVersion)
	if apiVersion == "" {
		return fmt.Errorf("missing apiVersion")
	}
	if apiVersion != expectedAPIVersion {
		return fmt.Errorf("apiVersion must be %q", expectedAPIVersion)
	}
	if strings.TrimSpace(kind) != expectedKind {
		return fmt.Errorf("kind must be %q", expectedKind)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("missing metadata.name")
	}
	if !RFC1123Label.MatchString(name) {
		return fmt.Errorf("metadata.name must adhere to rfc 1123: must be 1-63 characters, start with a lowercase letter, contain only lowercase letters, digits, or '-', and end with a letter or digit")
	}
	return nil
}

func (d ServiceDefinition) Validate() error {
	if err := validateApiObject(d.APIVersion, d.Kind, d.Metadata.Name, ServiceDefinitionAPIVersion, ServiceDefinitionKind); err != nil {
		return err
	}
	if strings.TrimSpace(d.Spec.ChartPath) == "" {
		return fmt.Errorf("missing spec.chartPath")
	}
	if d.Spec.Status != service.StatusEnabled && d.Spec.Status != service.StatusDisabled {
		return fmt.Errorf(`spec.status must be either %q or %q`, service.StatusEnabled, service.StatusDisabled)
	}

	return nil
}

func (m CatalogManifest) Validate() error {
	if err := validateApiObject(m.APIVersion, m.Kind, m.Metadata.Name, CatalogAPIVersion, CatalogKind); err != nil {
		return err
	}

	version := strings.TrimSpace(m.Spec.Version)
	if version == "" {
		return fmt.Errorf("missing spec.version")
	}
	if !StrictCatalogVersion.MatchString(version) {
		return fmt.Errorf(`spec.version must match exact semantic version format "x.y.z" without a leading "v"`)
	}

	return nil
}
