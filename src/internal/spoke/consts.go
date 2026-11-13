package spoke

const (
	Namespace = "kubara-system"

	SecretType             = "kubara.io/cluster-credentials"
	SecretSuffix           = "-cluster-secret"
	ConfigMapSuffix        = "-config"
	FluxSecretSuffix       = "-kubeconfig"
	BootstrapKubeconfigKey = "bootstrapKubeconfig"
	KubeconfigKey          = "kubeconfig"
	ArgoConfigKey          = "config"
	FluxKubeconfigKey      = "value"

	ClusterNameLabel         = "kubara.io/cluster-name"
	ArgoSecretTypeLabel      = "argocd.argoproj.io/secret-type"
	ArgoSecretTypeCluster    = "cluster"
	ManagedByAnnotation      = "kubara.io/cluster-credential-managed-by"
	ManagedByValue           = "cluster-credential-rotator"
	IssuedAtAnnotation       = "kubara.io/cluster-credential-issued-at"
	ExpiresAtAnnotation      = "kubara.io/cluster-credential-expires-at"
	RotationStatusAnnotation = "kubara.io/rotation-status"
	RotatedAtAnnotation      = "kubara.io/rotated-at"
	RotationErrorAnnotation  = "kubara.io/rotation-error"

	RotationStatusReady  = "ready"
	RotationStatusFailed = "failed"

	AgentServiceAccountName     = "kubara-agent"
	AgentClusterRoleBindingName = "kubara-agent"
	DefaultAgentClusterRoleName = "cluster-admin"
)

func SecretNameForCluster(clusterName string) string {
	return clusterName + SecretSuffix
}

func ConfigMapNameForCluster(clusterName string) string {
	return clusterName + ConfigMapSuffix
}
