package config

import (
	"slices"

	"github.com/kubara-io/kubara/internal/service"
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

// Config is the root of the configuration structure.
type Config struct {
	BootstrapCatalog *string   `json:"bootstrapCatalog,omitempty" yaml:"bootstrapCatalog,omitempty"`
	Clusters         []Cluster `json:"clusters" yaml:"clusters"`
}

// Cluster defines the configuration for a single Kubernetes cluster.
type Cluster struct {
	Name    string `json:"name"`
	Stage   string `json:"stage"`
	Type    string `json:"type"`
	DNSName string `json:"dnsName"`

	SSOOrg  string `json:"ssoOrg,omitempty"`
	SSOTeam string `json:"ssoTeam,omitempty"`

	IngressClassName string `json:"ingressClassName,omitempty"`

	Terraform *Terraform       `json:"terraform,omitempty"`
	ArgoCD    ArgoCD           `json:"argocd"`
	Catalogs  []string         `json:"catalogs,omitempty"`
	Services  service.Services `json:"services"`
}

type Terraform struct {
	Provider          TerraformProvider `json:"provider"`
	ProjectID         string            `json:"projectId"`
	KubernetesType    string            `json:"kubernetesType"`
	KubernetesVersion string            `json:"kubernetesVersion"`
	DNS               DNS               `json:"dns"`
}

type DNS struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ArgoCDSelfManagedStatus string

const (
	ArgoCDSelfManagedEnabled  ArgoCDSelfManagedStatus = "enabled"
	ArgoCDSelfManagedDisabled ArgoCDSelfManagedStatus = "disabled"
)

type ArgoCD struct {
	SelfManaged ArgoCDSelfManagedStatus `json:"selfManaged,omitempty"`
	Repo        RepoProto               `json:"repo"`
	HelmRepo    *HelmRepository         `json:"helmRepo,omitempty"`
}

type RepoProto struct {
	HTTPS *RepoType `json:"https,omitempty"`
	OCI   *RepoType `json:"oci,omitempty"`
}

type RepoType struct {
	Configs    Repository `json:"configs"`
	Components Repository `json:"components"`
}

type Repository struct {
	URL            string `json:"url"`
	Path           string `json:"path"`
	TargetRevision string `json:"targetRevision"`
}

type HelmRepository struct {
	URL string `json:"url"`
}
