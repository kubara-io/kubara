package v1alpha1

import (
	"github.com/kubara-io/kubara/internal/service"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Catalog describes a catalog root manifest.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=cat
type Catalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CatalogSpec `json:"spec"`
}

// CatalogSpec contains catalog-level settings.
type CatalogSpec struct {
	// +kubebuilder:validation:Pattern=`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`
	Version string `json:"version"`
}

// ServiceDefinition describes a catalog service entry.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=sd
type ServiceDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceSpec `json:"spec"`
}

// ServiceSpec contains the desired behavior and schema of a service.
type ServiceSpec struct {
	// ChartPath points to the immutable Helm chart path under platform-components/helm/.
	// +kubebuilder:validation:MinLength=1
	ChartPath string `json:"chartPath"`

	// Status defines the default status for the service and may be overridden per cluster.
	// +kubebuilder:validation:Enum=enabled;disabled
	Status service.Status `json:"status"`

	// ClusterTypes limits the service to specific cluster types as catalog metadata.
	// +optional
	ClusterTypes []string `json:"clusterTypes,omitempty"`

	// ConfigSchema describes config values using OpenAPI v3 schema props.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	ConfigSchema *apiextensionsv1.JSONSchemaProps `json:"configSchema,omitempty"`
}
