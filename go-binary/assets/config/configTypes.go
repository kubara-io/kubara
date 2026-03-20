package config

// Config is the root of the configuration structure.
type Config struct {
	Clusters []Cluster `json:"clusters" yaml:"clusters" jsonschema:"title=Clusters,description=A list of cluster configurations."`
}

// Cluster defines the configuration for a single Kubernetes cluster.
type Cluster struct {
	Name    string `json:"name" yaml:"name" jsonschema:"required,title=Cluster Name,description=The unique name for the cluster.,minLength=1,example=my-prod-cluster"`
	Stage   string `json:"stage" yaml:"stage" jsonschema:"required,title=Deployment Stage,description=The stage this cluster represents.,minLength=1,default=dev"`
	Type    string `json:"type" yaml:"type" jsonschema:"required,title=Cluster Type,description=The type of the cluster,enum=controlplane,enum=worker,default=controlplane"`
	DNSName string `json:"dnsName" yaml:"dnsName" jsonschema:"required,title=Primary DNS Name,description=The fully qualified domain name for the cluster.,format=hostname,example=my-prod-cluster.example.com"`

	SSOOrg  string `json:"ssoOrg,omitempty" yaml:"ssoOrg,omitempty" jsonschema:"title=SSO Organization,description=The SSO organization or group allowed to access this cluster.,minLength=1"`
	SSOTeam string `json:"ssoTeam,omitempty" yaml:"ssoTeam,omitempty" jsonschema:"title=SSO Team,description=The specific SSO team or sub-group allowed to access this cluster.,minLength=1"`

	IngressClassName string `json:"ingressClassName,omitempty" yaml:"ingressClassName,omitempty" jsonschema:"title=Ingress Class,description=The ingress class to use for this cluster.,minLength=1,default=traefik"`

	PrivateLoadBalancerIP string `json:"privateLoadBalancerIP,omitempty" yaml:"privateLoadBalancerIP,omitempty" jsonschema:"title=Private Load Balancer IP,description=The static IP for the private ingress controller load balancer.,format=ipv4"`
	PublicLoadBalancerIP  string `json:"publicLoadBalancerIP,omitempty" yaml:"publicLoadBalancerIP,omitempty" jsonschema:"title=Public Load Balancer IP,description=The static IP for the public ingress controller load balancer.,format=ipv4"`

	Terraform *Terraform `json:"terraform,omitempty" yaml:"terraform,omitempty" jsonschema:"title=Terraform,description=Configuration for terraform resources."`
	ArgoCD    ArgoCD     `json:"argocd" yaml:"argocd" jsonschema:"required,title=ArgoCD,description=Configuration for argoCD."`
	Services  Services   `json:"services" yaml:"services" jsonschema:"required,title=Services,description=Configuration for deployed services."`
}

type Terraform struct {
	ProjectID         string `json:"projectId" yaml:"projectId" jsonschema:"required,title=Cloud Project ID,description=The cloud provider project or subscription identifier. Accepts various formats depending on the provider.,minLength=1"`
	KubernetesType    string `json:"kubernetesType" yaml:"kubernetesType" jsonschema:"required,title=Kubernetes Type,description=The type of Kubernetes cluster.,enum=edge,enum=ske,default=ske"`
	KubernetesVersion string `json:"kubernetesVersion" yaml:"kubernetesVersion" jsonschema:"required,title=Kubernetes Version,description=The Kubernetes version for the cluster.,example=1.34,pattern=^[0-9]\\.[0-9]+(\\.[0-9]+)?$"`
	DNS               DNS    `json:"dns" yaml:"dns" jsonschema:"required,title=DNS Config,description=DNS Zone configuration"`
}

type DNS struct {
	Name  string `json:"name" yaml:"name" jsonschema:"required,title=DNS Zone Name,description=The managed DNS zone name.,format=hostname"`
	Email string `json:"email" yaml:"email" jsonschema:"required,title=Admin Email,description=Administrative email for the DNS zone.,format=email"`
}

type ArgoCD struct {
	Repo     RepoProto       `json:"repo" yaml:"repo" jsonschema:"required,title=ArgoCD Git Repository"`
	HelmRepo *HelmRepository `json:"helmRepo,omitempty" yaml:"helmRepo,omitempty" jsonschema:"title=ArgoCD Helm Charts Repository"`
}

type RepoProto struct {
	_     struct{}  `jsonschema:"minProperties=1,additionalProperties=false"`
	HTTPS *RepoType `json:"https,omitempty" yaml:"https,omitempty" jsonschema:"title=Https Repository"`
	OCI   *RepoType `json:"oci,omitempty" yaml:"oci,omitempty" jsonschema:"title=Oci Repository"`
}

type RepoType struct {
	Customer Repository `json:"customer" yaml:"customer" jsonschema:"required,title=Customer Repository"`
	Managed  Repository `json:"managed" yaml:"managed" jsonschema:"required,title=Managed Repository"`
}

type Repository struct {
	URL            string `json:"url" yaml:"url" jsonschema:"required,title=Repository URL,description=The HTTPS URL of the Git repository.,format=uri"`
	TargetRevision string `json:"targetRevision" yaml:"targetRevision" jsonschema:"required,title=Target Revision,description=The Git branch or tag to track.,minLength=1,default=main"`
}

type HelmRepository struct {
	URL string `json:"url" yaml:"url" jsonschema:"required,title=Repository URL,description=The Helm repository URL or OCI registry URL (without oci:// prefix),minLength=1"`
}

type ServiceStatus struct {
	Status Status `json:"status" yaml:"status" jsonschema:"required,title=Service Status,description=The desired status of the service.,enum=enabled,enum=disabled,default=disabled"`
}

type GenericService struct {
	ServiceStatus `yaml:",inline" jsonschema:"required,title=ServiceStatus"`
}

type Status string

const (
	StatusEnabled  Status = "enabled"
	StatusDisabled Status = "disabled"
)

// ClusterIssuer defines the nested configuration for cert-manager's issuer.
type ClusterIssuer struct {
	Name   string `json:"name" yaml:"name" jsonschema:"required,title=Issuer Name,description=Name of the ClusterIssuer resource.,minLength=1,example=letsencrypt-staging,default=letsencrypt-staging"`
	Email  string `json:"email" yaml:"email" jsonschema:"required,title=ACME Email,description=The email address for Let's Encrypt registration.,format=email"`
	Server string `json:"server" yaml:"server" jsonschema:"required,title=ACME Server,description=The ACME server URL.,format=uri,default=https://acme-staging-v02.api.letsencrypt.org/directory"`
}

// CertManagerService defines the specific, nested configuration for cert-manager.
type CertManagerService struct {
	ServiceStatus `yaml:",inline"`
	ClusterIssuer ClusterIssuer `json:"clusterIssuer" yaml:"clusterIssuer" jsonschema:"required,title=Cluster Issuer Configuration"`
}

type Services struct {
	Argocd              GenericService     `json:"argocd" yaml:"argocd" jsonschema:"required,title=Argocd Service,description=Configuration for argoCD"`
	CertManager         CertManagerService `json:"certManager" yaml:"certManager" jsonschema:"required,title=CertManager Service,description=Configuration for CertManager"`
	ExternalDns         GenericService     `json:"externalDns" yaml:"externalDns" jsonschema:"required,title=ExternalDns Service,description=Configuration for ExternalDns"`
	ExternalSecrets     GenericService     `json:"externalSecrets" yaml:"externalSecrets" jsonschema:"required,title=ExternalSecrets Service,description=Configuration for ExternalSecrets"`
	KubePrometheusStack GenericService     `json:"kubePrometheusStack" yaml:"kubePrometheusStack" jsonschema:"required,title=KubePrometheusStack Service,description=Configuration for KubePrometheusStack"`
	Traefik             GenericService     `json:"traefik" yaml:"traefik" jsonschema:"required,title=Traefik Service,description=Configuration for Traefik"`
	Kyverno             GenericService     `json:"kyverno" yaml:"kyverno" jsonschema:"required,title=Kyverno Service,description=Configuration for Kyverno"`
	KyvernoPolicies     GenericService     `json:"kyvernoPolicies" yaml:"kyvernoPolicies" jsonschema:"required,title=KyvernoPolicies Service,description=Configuration for KyvernoPolicies"`
	KyvernoPolicyReport GenericService     `json:"kyvernoPolicyReport" yaml:"kyvernoPolicyReport" jsonschema:"required,title=KyvernoPolicyReport Service,description=Configuration for KyvernoPolicyReport"`
	Loki                GenericService     `json:"loki" yaml:"loki" jsonschema:"required,title=Loki Service,description=Configuration for Loki"`
	HomerDashboard      GenericService     `json:"homerDashboard" yaml:"homerDashboard" jsonschema:"required,title=HomerDashboard Service,description=Configuration for HomerDashboard"`
	Oauth2Proxy         GenericService     `json:"oauth2Proxy" yaml:"oauth2Proxy" jsonschema:"required,title=oauth2Proxy Service,description=Configuration for oaOauth2Proxy"`
	MetricsServer       GenericService     `json:"metricsServer" yaml:"metricsServer" jsonschema:"required,title=MetricsServer Service,description=Configuration for MetricsServer"`
	MetalLb             GenericService     `json:"metalLb" yaml:"metalLb" jsonschema:"required,title=metalLb Service,description=Configuration for metalLb"`
	Longhorn            GenericService     `json:"longhorn" yaml:"longhorn" jsonschema:"required,title=Longhorn Service,description=Configuration for Longhorn"`
}
