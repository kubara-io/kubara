package config

import (
	"slices"

	"github.com/kubara-io/kubara/internal/service"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Hub   string = "hub"
	Spoke string = "spoke"
)

// TerraformProvider identifies an infrastructure provider supported by catalog generation.
type TerraformProvider string

const (
	TerraformProviderNone         TerraformProvider = "none"
	TerraformProviderStackit      TerraformProvider = "stackit"
	TerraformProviderTCloudPublic TerraformProvider = "t-cloud-public"
)

var supportedTerraformProviders = [...]TerraformProvider{
	TerraformProviderStackit,
	TerraformProviderTCloudPublic,
}

// IsSupported reports whether kubara supports Terraform generation for the provider.
func (p TerraformProvider) IsSupported() bool {
	return slices.Contains(supportedTerraformProviders[:], p)
}

// SupportedTerraformProviders returns the providers supported by Terraform generation.
func SupportedTerraformProviders() []TerraformProvider {
	return append([]TerraformProvider(nil), supportedTerraformProviders[:]...)
}

// PlatformSetup is the Schema for the platformsetups API.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,shortName=ps
type PlatformSetup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec Config `json:"spec"`
}

// Config is the root of the configuration structure.
type Config struct {
	// Global bootstrap catalog reference.
	// +optional
	BootstrapCatalog *string `json:"bootstrapCatalog,omitempty" yaml:"bootstrapCatalog,omitempty"`

	// +listType=map
	// +listMapKey=name
	Clusters []Cluster `json:"clusters" yaml:"clusters"`
}

// Cluster defines the configuration for a single Kubernetes cluster.
type Cluster struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// +kubebuilder:default=dev
	// +kubebuilder:validation:MinLength=1
	// +optional
	Stage string `json:"stage,omitempty"`

	// +kubebuilder:default=hub
	// +kubebuilder:validation:Enum=hub;spoke
	// +optional
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Format=hostname
	DNSName string `json:"dnsName"`

	// +kubebuilder:validation:MinLength=1
	// +optional
	SSOOrg string `json:"ssoOrg,omitempty"`

	// +kubebuilder:validation:MinLength=1
	// +optional
	SSOTeam string `json:"ssoTeam,omitempty"`

	// +kubebuilder:default=traefik
	// +kubebuilder:validation:MinLength=1
	// +optional
	IngressClassName string `json:"ingressClassName,omitempty"`

	// +optional
	Terraform *Terraform `json:"terraform,omitempty"`

	ArgoCD ArgoCD `json:"argocd"`

	// +optional
	Catalogs []string `json:"catalogs,omitempty"`

	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Services service.Services `json:"services"`
}

// +kubebuilder:validation:XValidation:rule="self.provider == 'none' || (has(self.projectId) && has(self.kubernetesVersion) && has(self.dns))",message="projectId, kubernetesVersion, and dns are required when terraform.provider is enabled"
// +kubebuilder:validation:XValidation:rule="self.provider != 'stackit' || self.kubernetesType in ['ske', 'edge']",message="stackit supports kubernetesType ske or edge"
// +kubebuilder:validation:XValidation:rule="self.provider != 't-cloud-public' || self.kubernetesType == 'cce'",message="t-cloud-public supports kubernetesType cce"
type Terraform struct {
	// +kubebuilder:default=none
	// +kubebuilder:validation:Enum=none;stackit;t-cloud-public
	Provider TerraformProvider `json:"provider"`

	// +kubebuilder:validation:MinLength=1
	// +optional
	ProjectID string `json:"projectId,omitempty"`

	// +kubebuilder:default=ske
	// +kubebuilder:validation:Enum=edge;ske;cce
	// +optional
	KubernetesType string `json:"kubernetesType,omitempty"`

	// +kubebuilder:validation:Pattern=`^[0-9]\.[0-9]+(\.[0-9]+)?$`
	// +optional
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`

	// +optional
	DNS DNS `json:"dns,omitempty"`
}

type DNS struct {
	// +kubebuilder:validation:Format=hostname
	Name string `json:"name"`

	// +kubebuilder:validation:Format=email
	Email string `json:"email"`
}

type ArgoCDSelfManagedStatus string

const (
	ArgoCDSelfManagedEnabled  ArgoCDSelfManagedStatus = "enabled"
	ArgoCDSelfManagedDisabled ArgoCDSelfManagedStatus = "disabled"
)

type ArgoCD struct {
	// +kubebuilder:default=enabled
	// +kubebuilder:validation:Enum=enabled;disabled
	// +optional
	SelfManaged ArgoCDSelfManagedStatus `json:"selfManaged,omitempty"`

	Repo RepoProto `json:"repo"`

	// +optional
	HelmRepo *HelmRepository `json:"helmRepo,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.https) != has(self.oci)",message="exactly one of https or oci must be configured"
type RepoProto struct {
	// +optional
	HTTPS *RepoType `json:"https,omitempty"`

	// +optional
	OCI *RepoType `json:"oci,omitempty"`
}

type RepoType struct {
	Configs    Repository `json:"configs"`
	Components Repository `json:"components"`
}

type Repository struct {
	// +kubebuilder:validation:Format=uri
	URL string `json:"url"`

	// +optional
	Path string `json:"path,omitempty"`

	// +kubebuilder:default=main
	// +kubebuilder:validation:MinLength=1
	// +optional
	TargetRevision string `json:"targetRevision,omitempty"`
}

type HelmRepository struct {
	// +kubebuilder:validation:Format=uri
	URL string `json:"url"`
}
