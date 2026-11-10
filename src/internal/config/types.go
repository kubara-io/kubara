package config

import (
	"encoding/json"
	"slices"

	"github.com/kubara-io/kubara/internal/service"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ConfigVersionV1Alpha1 = "v1alpha1"
	ConfigVersionV1Alpha2 = "v1alpha2"
	ConfigVersionV1Alpha3 = "v1alpha3"
	ConfigVersionV1Alpha4 = "v1alpha4"
	ConfigVersionV1Alpha5 = "v1alpha5"
	ConfigVersionV1Alpha6 = "v1alpha6"
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

	// Spec defines the desired state of the platform setup configuration.
	Spec Config `json:"spec"`
}

// Config is the root of the configuration structure.
// +kubebuilder:validation:XValidation:rule="self.clusters.size() == 0 || self.clusters.filter(c, c.type == 'hub').size() == 1",message="exactly one cluster must be of type hub"
type Config struct {
	// BootstrapCatalog is the global bootstrap catalog reference.
	// +kubebuilder:validation:MinLength=1
	// +optional
	BootstrapCatalog *string `json:"bootstrapCatalog,omitempty" yaml:"bootstrapCatalog,omitempty"`

	// Clusters defines the list of Kubernetes clusters managed by the platform.
	// +listType=map
	// +listMapKey=name
	Clusters []Cluster `json:"clusters" yaml:"clusters"`
}

// Cluster defines the configuration for a single Kubernetes cluster.
type Cluster struct {
	// Name is the unique identifier for the cluster.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Type specifies whether this cluster is a hub or spoke cluster.
	// +kubebuilder:default=hub
	// +kubebuilder:validation:Enum=hub;spoke
	// +optional
	Type string `json:"type,omitempty"`

	// Stage is the deployment environment stage (e.g., dev, staging, prod).
	// +kubebuilder:default=dev
	// +kubebuilder:validation:MinLength=1
	// +optional
	Stage string `json:"stage,omitempty"`

	// Deprecated: use Networking.Ingress.ClassName. Retained for older catalogs.
	IngressClassName string `json:"-" yaml:"-"`

	// Networking configures routing settings consumed by catalog templates.
	// +optional
	Networking ClusterNetworking `json:"networking,omitempty"`

	// Image Pull Secret Name,description=Shared registry secret name.
	// +kubebuilder:validation:Pattern=`^$|^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MaxLength=249
	// +optional
	ImagePullSecretName *string `json:"imagePullSecretName,omitempty"`

	// DNSName is the primary domain name for the cluster.
	// +kubebuilder:validation:Format=hostname
	DNSName string `json:"dnsName"`

	// SSOOrg is the single sign-on organization identifier for access control.
	// +kubebuilder:validation:MinLength=1
	// +optional
	SSOOrg string `json:"ssoOrg,omitempty"`

	// SSOTeam is the single sign-on team identifier within the organization.
	// +kubebuilder:validation:MinLength=1
	// +optional
	SSOTeam string `json:"ssoTeam,omitempty"`

	// Terraform configures infrastructure provisioning for the cluster.
	// +optional
	Terraform *Terraform `json:"terraform,omitempty"`

	// ArgoCD configures the Argo CD deployment and repository settings for the cluster.
	ArgoCD ArgoCD `json:"argocd"`

	// Catalogs specifies additional catalog references available to this cluster.
	// +optional
	Catalogs []string `json:"catalogs,omitempty"`

	// Services configures the platform services deployed on this cluster.
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:pruning:PreserveUnknownFields
	Services service.Services `json:"services"`
}

func (c Cluster) MarshalJSON() ([]byte, error) {
	type Alias Cluster
	return json.Marshal(&struct {
		Alias
		IngressClassName string `json:"ingressClassName,omitempty"`
	}{
		Alias:            Alias(c),
		IngressClassName: c.IngressClassName,
	})
}

// DefaultImagePullSecretName defaults an non-existend ImagePullSecretName field
// to the default value "image-pull-secret" but differentiates between non-existens
// and empty/zero value "" and returns an empty string if explicitely set by the user
func (c *Cluster) DefaultImagePullSecretName() string {
	if c.ImagePullSecretName == nil {
		return "image-pull-secret"
	}
	return *c.ImagePullSecretName
}

// ClusterNetworking contains the shared routing inputs for catalog templates.
// +kubebuilder:validation:XValidation:rule="!has(self.type) || self.type != 'gateway' || has(self.gateway)",message="networking.gateway is required when type is gateway"
type ClusterNetworking struct {
	// Type selects the routing API (Ingress or Gateway API) used by catalog templates.
	// +kubebuilder:validation:Enum=ingress;gateway
	// +kubebuilder:default=ingress
	// +optional
	Type NetworkingType `json:"type,omitempty"`

	// Ingress configures global Ingress settings.
	// +optional
	Ingress *IngressNetworking `json:"ingress,omitempty"`

	// Gateway configures global Gateway settings.
	// +optional
	Gateway *service.GatewayReference `json:"gateway,omitempty"`
}

// NetworkingType selects the routing API.
type NetworkingType string

const (
	NetworkingIngress NetworkingType = "ingress"
	NetworkingGateway NetworkingType = "gateway"
)

// IngressNetworking contains Ingress-specific routing settings.
type IngressNetworking struct {
	// ClassName is the Ingress class name to use for routing resources.
	// +kubebuilder:default=traefik
	// +kubebuilder:validation:MinLength=1
	// +optional
	ClassName string `json:"className,omitempty"`
}

// Terraform configures infrastructure provisioning for a cluster.
// +kubebuilder:validation:XValidation:rule="self.provider == 'none' || (has(self.projectId) && has(self.kubernetesVersion) && has(self.dns))",message="projectId, kubernetesVersion, and dns are required when terraform.provider is enabled"
// +kubebuilder:validation:XValidation:rule="self.provider != 'stackit' || self.kubernetesType in ['ske', 'edge']",message="stackit supports kubernetesType ske or edge"
// +kubebuilder:validation:XValidation:rule="self.provider != 't-cloud-public' || self.kubernetesType == 'cce'",message="t-cloud-public supports kubernetesType cce"
type Terraform struct {
	// Provider specifies the infrastructure provider to use for cluster provisioning.
	// +kubebuilder:default=none
	// +kubebuilder:validation:Enum=none;stackit;t-cloud-public
	Provider TerraformProvider `json:"provider"`

	// ProjectID is the cloud provider project or tenant identifier.
	// +kubebuilder:validation:MinLength=1
	// +optional
	ProjectID string `json:"projectId,omitempty"`

	// KubernetesType specifies the managed Kubernetes offering (e.g., ske, edge, cce).
	// +kubebuilder:default=ske
	// +kubebuilder:validation:Enum=edge;ske;cce
	// +optional
	KubernetesType string `json:"kubernetesType,omitempty"`

	// KubernetesVersion specifies the target Kubernetes version (e.g., 1.30 or 1.30.1).
	// +kubebuilder:validation:Pattern=`^[0-9]\.[0-9]+(\.[0-9]+)?$`
	// +optional
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`

	// DNS configures the DNS settings for the infrastructure provider.
	// +optional
	DNS DNS `json:"dns,omitempty"`
}

// DNS defines domain name and contact information for DNS management.
type DNS struct {
	// Name is the managed DNS zone or domain name.
	// +kubebuilder:validation:Format=hostname
	Name string `json:"name"`

	// Email is the administrator contact email for DNS records.
	// +kubebuilder:validation:Format=email
	Email string `json:"email"`
}

type ArgoCDSelfManagedStatus string

const (
	ArgoCDSelfManagedEnabled  ArgoCDSelfManagedStatus = "enabled"
	ArgoCDSelfManagedDisabled ArgoCDSelfManagedStatus = "disabled"
)

type GitAuthMode string

const (
	GitAuthModeHTTPS     GitAuthMode = "https"
	GitAuthModeSSH       GitAuthMode = "ssh"
	GitAuthModeGitHubApp GitAuthMode = "github-app"
)

// ArgoCD defines configuration for Argo CD cluster integration and repository tracking.
type ArgoCD struct {
	// SelfManaged indicates whether Argo CD should be self-managed by the platform.
	// +kubebuilder:default=enabled
	// +kubebuilder:validation:Enum=enabled;disabled
	// +optional
	SelfManaged ArgoCDSelfManagedStatus `json:"selfManaged,omitempty"`

	// Repo configures the Git or OCI repository sources for Argo CD applications.
	Repo RepoProto `json:"repo"`

	// HelmRepo configures an optional Helm chart repository for Argo CD.
	// +optional
	HelmRepo *HelmRepository `json:"helmRepo,omitempty"`
}

// RepoProto configures repository protocol sources and authentication for Argo CD.
// +kubebuilder:validation:XValidation:rule="has(self.git) != has(self.oci)",message="exactly one of git or oci must be configured"
type RepoProto struct {
	// AuthMode specifies the authentication mechanism used for Git operations.
	// +kubebuilder:validation:Enum=https;ssh;github-app
	// +optional
	AuthMode GitAuthMode `json:"authMode,omitempty"`

	// Git configures Git repositories for platform configs and components.
	// +optional
	Git *RepoType `json:"git,omitempty"`

	// OCI configures OCI registry repositories for platform configs and components.
	// +optional
	OCI *RepoType `json:"oci,omitempty"`
}

// RepoType specifies the repository targets for platform configurations and components.
type RepoType struct {
	// Configs is the repository containing platform configuration definitions.
	Configs Repository `json:"configs"`

	// Components is the repository containing platform component manifests.
	Components Repository `json:"components"`
}

// Repository defines a source repository location, target path, and revision.
type Repository struct {
	// URL is the repository address (e.g. HTTPS or SSH URL).
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`

	// Path is an optional subdirectory path within the repository.
	// +optional
	Path string `json:"path,omitempty"`

	// TargetRevision is the branch, tag, or commit to track.
	// +kubebuilder:default=main
	// +kubebuilder:validation:MinLength=1
	// +optional
	TargetRevision string `json:"targetRevision,omitempty"`
}

// HelmRepository defines configuration for an external Helm chart repository.
type HelmRepository struct {
	// URL is the Helm chart repository URL.
	// +kubebuilder:validation:Format=uri
	URL string `json:"url"`
}
