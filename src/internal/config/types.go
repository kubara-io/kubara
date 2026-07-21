package config

import (
	"slices"

	"github.com/kubara-io/kubara/internal/service"
)

const (
	ConfigVersionV1Alpha1 = "v1alpha1"
	ConfigVersionV1Alpha2 = "v1alpha2"
	ConfigVersionV1Alpha3 = "v1alpha3"
	ConfigVersionV1Alpha4 = "v1alpha4"
)

const (
	Hub   string = "hub"
	Spoke string = "spoke"
)

// TerraformProvider identifies an infrastructure provider with embedded Terraform templates.
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

// IsSupported reports whether kubara ships Terraform templates for the provider.
func (p TerraformProvider) IsSupported() bool {
	return slices.Contains(supportedTerraformProviders[:], p)
}

// SupportedTerraformProviders returns the providers with embedded Terraform templates.
func SupportedTerraformProviders() []TerraformProvider {
	return append([]TerraformProvider(nil), supportedTerraformProviders[:]...)
}

// Config is the root of the configuration structure.
type Config struct {
	Version  string    `json:"version,omitempty" yaml:"version,omitempty" jsonschema:"title=Config Version,description=The schema version of this config file.,enum=v1alpha4,default=v1alpha4"`
	Clusters []Cluster `json:"clusters" yaml:"clusters" jsonschema:"title=Clusters,description=A list of cluster configurations."`
}

// Cluster defines the configuration for a single Kubernetes cluster.
type Cluster struct {
	Name    string `json:"name" yaml:"name" jsonschema:"required,title=Cluster Name,description=The unique name for the cluster.,minLength=1,example=my-prod-cluster"`
	Stage   string `json:"stage" yaml:"stage" jsonschema:"title=Deployment Stage,description=The stage this cluster represents.,minLength=1,default=dev"`
	Type    string `json:"type" yaml:"type" jsonschema:"title=Cluster Type,description=The type of the cluster,enum=hub,enum=spoke,default=hub"`
	DNSName string `json:"dnsName" yaml:"dnsName" jsonschema:"required,title=Primary DNS Name,description=The fully qualified domain name for the cluster.,format=hostname,example=my-prod-cluster.example.com"`

	SSOOrg  string `json:"ssoOrg,omitempty" yaml:"ssoOrg,omitempty" jsonschema:"title=SSO Organization,description=The SSO organization or group allowed to access this cluster.,minLength=1"`
	SSOTeam string `json:"ssoTeam,omitempty" yaml:"ssoTeam,omitempty" jsonschema:"title=SSO Team,description=The specific SSO team or sub-group allowed to access this cluster.,minLength=1"`

	IngressClassName string `json:"ingressClassName,omitempty" yaml:"ingressClassName,omitempty" jsonschema:"title=Ingress Class,description=The ingress class to use for this cluster.,minLength=1,default=traefik"`

	Terraform *Terraform       `json:"terraform,omitempty" yaml:"terraform,omitempty" jsonschema:"title=Terraform,description=Configuration for terraform resources."`
	ArgoCD    ArgoCD           `json:"argocd" yaml:"argocd" jsonschema:"required,title=ArgoCD,description=Configuration for argoCD."`
	Services  service.Services `json:"services" yaml:"services" jsonschema:"required,title=Services,description=Configuration for deployed services."`
}

type Terraform struct {
	Provider          TerraformProvider `json:"provider" yaml:"provider" jsonschema:"title=Cloud Provider,description=Infrastructure provider used for Terraform templates. Use none to skip Terraform generation. Currently supported providers: stackit and t-cloud-public.,enum=none,enum=stackit,enum=t-cloud-public,default=none"`
	ProjectID         string            `json:"projectId" yaml:"projectId" jsonschema:"required,title=Cloud Project ID,description=The provider-specific project subscription or tenant identifier. For t-cloud-public use the tenant or project name rather than a UUID.,minLength=1"`
	KubernetesType    string            `json:"kubernetesType" yaml:"kubernetesType" jsonschema:"title=Kubernetes Type,description=The type of Kubernetes cluster.,enum=edge,enum=ske,enum=cce,default=ske"`
	KubernetesVersion string            `json:"kubernetesVersion" yaml:"kubernetesVersion" jsonschema:"required,title=Kubernetes Version,description=The Kubernetes version for the cluster.,example=1.34,pattern=^[0-9]\\.[0-9]+(\\.[0-9]+)?$"`
	DNSContactEmail   string            `json:"dnsContactEmail" yaml:"dnsContactEmail" jsonschema:"required,title=DNS Zone Contact Email,description=Administrative contact email for the managed DNS zone. The zone name itself is derived from the cluster dnsName.,format=email"`
}

type ArgoCD struct {
	Repo     RepoProto       `json:"repo" yaml:"repo" jsonschema:"required,title=ArgoCD Git Repository"`
	HelmRepo *HelmRepository `json:"helmRepo,omitempty" yaml:"helmRepo,omitempty" jsonschema:"title=ArgoCD Helm Charts Repository"`
}

type RepoProto struct {
	_        struct{}  `jsonschema:"minProperties=1,additionalProperties=false"`
	AuthMode string    `json:"authMode,omitempty" yaml:"authMode,omitempty" jsonschema:"title=Git Auth Mode,description=Authentication mode kubara uses for the initial Argo CD Git repository secret.,enum=https,enum=ssh,enum=github-app,default=https"`
	Git      *RepoType `json:"git" yaml:"git" jsonschema:"required,title=Git Repository"`
	OCI      *RepoType `json:"oci,omitempty" yaml:"oci,omitempty" jsonschema:"title=Oci Repository"`
}

type RepoType struct {
	Configs    Repository `json:"configs" yaml:"configs" jsonschema:"required,title=Platform Configs Repository"`
	Components Repository `json:"components" yaml:"components" jsonschema:"required,title=Platform Components Repository"`
}

type Repository struct {
	URL            string `json:"url" yaml:"url" jsonschema:"required,title=Repository URL,description=The Git repository URL used by Argo CD. Use an HTTP(S) URL for https/github-app auth modes or an SSH URL for ssh auth mode.,minLength=1"`
	TargetRevision string `json:"targetRevision" yaml:"targetRevision" jsonschema:"title=Target Revision,description=The Git branch or tag to track.,minLength=1,default=main"`
}

type HelmRepository struct {
	URL string `json:"url" yaml:"url" jsonschema:"required,title=Repository URL,description=The Helm repository URL or OCI registry URL (without oci:// prefix),minLength=1"`
}
