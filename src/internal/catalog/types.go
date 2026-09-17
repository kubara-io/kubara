package catalog

import (
	"regexp"

	"github.com/kubara-io/kubara/internal/catalog/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// Type aliases for v1alpha1 API types
type ServiceDefinition = v1alpha1.ServiceDefinition
type ServiceSpec = v1alpha1.ServiceSpec
type CatalogManifest = v1alpha1.Catalog
type CatalogSpec = v1alpha1.CatalogSpec

// Metadata is an alias for metav1.ObjectMeta for backwards compatibility.
type Metadata = metav1.ObjectMeta

// Catalog represents a set of service definitions keyed by canonical service name.
type Catalog struct {
	// Services maps canonical service names to definitions.
	Services map[string]ServiceDefinition
}
