package spoke

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type SpokeClientFactory func(kubeconfig []byte) (kubernetes.Interface, error)

type Config struct {
	TokenDuration  time.Duration
	RefreshBefore  time.Duration
	ClusterRole    string
	ArgoEnabled    bool
	ArgoNamespace  string
	FluxEnabled    bool
	FluxNamespace  string
	Now            func() time.Time
	SpokeClientFor SpokeClientFactory
}

type Reconciler struct {
	hubClient kubernetes.Interface
	cfg       Config
}

func NewReconciler(hubClient kubernetes.Interface, cfg Config) (*Reconciler, error) {
	if hubClient == nil {
		return nil, fmt.Errorf("hub kubernetes client is required")
	}
	if cfg.TokenDuration <= 0 {
		return nil, fmt.Errorf("token duration must be greater than zero")
	}
	if cfg.RefreshBefore < 0 {
		return nil, fmt.Errorf("refreshBefore must not be negative")
	}
	if cfg.RefreshBefore >= cfg.TokenDuration {
		return nil, fmt.Errorf("refreshBefore (%s) must be smaller than token duration (%s)", cfg.RefreshBefore, cfg.TokenDuration)
	}
	if strings.TrimSpace(cfg.ClusterRole) == "" {
		cfg.ClusterRole = DefaultAgentClusterRoleName
	}
	if strings.TrimSpace(cfg.ArgoNamespace) == "" {
		cfg.ArgoNamespace = "argocd"
	}
	if strings.TrimSpace(cfg.FluxNamespace) == "" {
		cfg.FluxNamespace = "flux-system"
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.SpokeClientFor == nil {
		cfg.SpokeClientFor = DefaultSpokeClientFactory
	}

	return &Reconciler{
		hubClient: hubClient,
		cfg:       cfg,
	}, nil
}

func DefaultSpokeClientFactory(kubeconfig []byte) (kubernetes.Interface, error) {
	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("build spoke REST config: %w", err)
	}
	restCfg.Timeout = 30 * time.Second
	return kubernetes.NewForConfig(restCfg)
}

func (r *Reconciler) ReconcileAll(ctx context.Context) error {
	secretList, err := r.hubClient.CoreV1().
		Secrets(Namespace).
		List(ctx, metav1.ListOptions{
			FieldSelector: "type=" + SecretType,
		})
	if err != nil {
		return fmt.Errorf("list cluster credential secrets in namespace %q: %w", Namespace, err)
	}

	if len(secretList.Items) == 0 {
		log.Info().Msgf("No cluster credential secrets found in namespace %q", Namespace)
		return nil
	}

	names := make([]string, 0, len(secretList.Items))
	for _, item := range secretList.Items {
		names = append(names, item.Name)
	}
	sort.Strings(names)

	failureCount := 0
	for _, secretName := range names {
		if err := r.reconcileSecret(ctx, secretName); err != nil {
			failureCount++
			log.Error().Err(err).Str("secret", secretName).Msg("failed to reconcile cluster credential secret")
			r.recordRotationFailure(ctx, secretName, err.Error())
		}
	}

	if failureCount > 0 {
		return fmt.Errorf("%d secret reconciliation(s) failed", failureCount)
	}

	log.Info().Msg("Cluster credential rotation completed successfully")
	return nil
}

func (r *Reconciler) reconcileSecret(ctx context.Context, secretName string) error {
	secrets := r.hubClient.CoreV1().Secrets(Namespace)
	secret, err := secrets.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("read secret %q: %w", secretName, err)
	}

	if string(secret.Type) != SecretType {
		return nil
	}

	clusterName := strings.TrimSpace(string(secret.Data["name"]))
	if clusterName == "" {
		clusterName = strings.TrimSuffix(secretName, SecretSuffix)
	}

	configMapName := ConfigMapNameForCluster(clusterName)
	configMap, err := r.hubClient.CoreV1().
		ConfigMaps(Namespace).
		Get(ctx, configMapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return r.decommissionSpoke(ctx, secretName, clusterName)
	}
	if err != nil {
		return fmt.Errorf("get configmap %q: %w", configMapName, err)
	}

	if configMap.Labels[ClusterNameLabel] != clusterName {
		return fmt.Errorf("configmap %q has invalid %s label %q (expected %q)", configMapName, ClusterNameLabel, configMap.Labels[ClusterNameLabel], clusterName)
	}

	desiredLabels := maps.Clone(configMap.Labels)
	if desiredLabels == nil {
		desiredLabels = map[string]string{}
	}

	now := r.cfg.Now()
	currentKubeconfig := secret.Data[KubeconfigKey]

	var (
		server    string
		token     string
		caData    string
		insecure  bool
		issuedAt  int64
		expiresAt int64
	)

	if len(currentKubeconfig) == 0 {
		bootstrapKubeconfig := secret.Data[BootstrapKubeconfigKey]
		if len(bootstrapKubeconfig) == 0 {
			return fmt.Errorf("secret %q is missing required bootstrap key %q", secretName, BootstrapKubeconfigKey)
		}

		server, caData, insecure, err = extractClusterEndpoint(bootstrapKubeconfig)
		if err != nil {
			return fmt.Errorf("extract cluster endpoint from bootstrap kubeconfig for %q: %w", clusterName, err)
		}

		spokeClient, err := r.cfg.SpokeClientFor(bootstrapKubeconfig)
		if err != nil {
			return fmt.Errorf("connect to spoke %q with bootstrap kubeconfig: %w", clusterName, err)
		}

		if err := ensureSpokeAgent(ctx, spokeClient, r.cfg.ClusterRole); err != nil {
			return fmt.Errorf("provision kubara-agent on spoke %q: %w", clusterName, err)
		}

		token, expiresAt, err = mintSpokeAgentToken(ctx, spokeClient, r.cfg.TokenDuration)
		if err != nil {
			return fmt.Errorf("mint initial kubara-agent token for %q: %w", clusterName, err)
		}

		issuedAt = now.Unix()
		currentKubeconfig, err = buildAgentKubeconfig(server, caData, insecure, token)
		if err != nil {
			return fmt.Errorf("build initial kubeconfig for %q: %w", clusterName, err)
		}

		if err := r.validateSpokeKubeconfig(ctx, currentKubeconfig); err != nil {
			return fmt.Errorf("initial generated credentials for %q failed validation: %w", clusterName, err)
		}

		secret, err = r.persistHubCredentials(ctx, secret, desiredLabels, clusterName, server, currentKubeconfig, token, caData, insecure, issuedAt, expiresAt)
		if err != nil {
			return err
		}
	} else {
		if !maps.Equal(secret.Labels, desiredLabels) {
			updated := secret.DeepCopy()
			updated.Labels = maps.Clone(desiredLabels)
			secret, err = secrets.Update(ctx, updated, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("sync labels on secret %q: %w", secretName, err)
			}
		}

		server = strings.TrimSpace(string(secret.Data["server"]))
		token = strings.TrimSpace(string(secret.Data["token"]))
		caData = strings.TrimSpace(string(secret.Data["caData"]))
		insecure = strings.TrimSpace(string(secret.Data["insecure"])) == "true"

		if err := r.validateSpokeKubeconfig(ctx, currentKubeconfig); err != nil {
			return fmt.Errorf("current credentials for %q are invalid; automatic rotation cannot continue without explicit re-onboarding: %w", clusterName, err)
		}

		expiresAt, _ = strconv.ParseInt(secret.Annotations[ExpiresAtAnnotation], 10, 64)
		issuedAt, _ = strconv.ParseInt(secret.Annotations[IssuedAtAnnotation], 10, 64)

		refreshHorizon := now.Add(r.cfg.RefreshBefore).Unix()
		if expiresAt <= 0 || refreshHorizon >= expiresAt {
			spokeClient, err := r.cfg.SpokeClientFor(currentKubeconfig)
			if err != nil {
				return fmt.Errorf("connect to spoke %q for rotation: %w", clusterName, err)
			}

			token, expiresAt, err = mintSpokeAgentToken(ctx, spokeClient, r.cfg.TokenDuration)
			if err != nil {
				return fmt.Errorf("mint replacement kubara-agent token for %q: %w", clusterName, err)
			}

			issuedAt = now.Unix()
			if expiresAt <= issuedAt {
				return fmt.Errorf("token request returned an already-expired credential for %q", clusterName)
			}
			actualLifetime := time.Duration(expiresAt-issuedAt) * time.Second
			if actualLifetime <= r.cfg.RefreshBefore {
				return fmt.Errorf("token lifetime returned for %q (%s) is not greater than refreshBefore (%s)", clusterName, actualLifetime, r.cfg.RefreshBefore)
			}

			currentKubeconfig, err = buildAgentKubeconfig(server, caData, insecure, token)
			if err != nil {
				return fmt.Errorf("build replacement kubeconfig for %q: %w", clusterName, err)
			}

			if err := r.validateSpokeKubeconfig(ctx, currentKubeconfig); err != nil {
				return fmt.Errorf("replacement credentials for %q failed validation: %w", clusterName, err)
			}

			secret, err = r.persistHubCredentials(ctx, secret, desiredLabels, clusterName, server, currentKubeconfig, token, caData, insecure, issuedAt, expiresAt)
			if err != nil {
				return err
			}
		}
	}

	if issuedAt == 0 {
		issuedAt = now.Unix()
	}

	if r.cfg.ArgoEnabled {
		if err := r.projectArgoTarget(ctx, clusterName, server, token, caData, insecure, desiredLabels, issuedAt, expiresAt); err != nil {
			return err
		}
	}

	if r.cfg.FluxEnabled {
		if err := r.projectFluxTarget(ctx, clusterName, currentKubeconfig, desiredLabels, issuedAt, expiresAt); err != nil {
			return err
		}
	}

	return r.markSecretReady(ctx, secret, issuedAt)
}

func (r *Reconciler) decommissionSpoke(ctx context.Context, secretName, clusterName string) error {
	log.Info().Str("cluster", clusterName).Msg("Decommissioning spoke cluster (authoritative ConfigMap NotFound)")

	if r.cfg.ArgoEnabled {
		if err := r.deleteManagedTargetSecret(ctx, r.cfg.ArgoNamespace, SecretNameForCluster(clusterName)); err != nil {
			return fmt.Errorf("delete managed Argo CD target secret for %q: %w", clusterName, err)
		}
	}

	if r.cfg.FluxEnabled {
		if err := r.deleteManagedTargetSecret(ctx, r.cfg.FluxNamespace, clusterName+FluxSecretSuffix); err != nil {
			return fmt.Errorf("delete managed Flux target secret for %q: %w", clusterName, err)
		}
	}

	if err := r.hubClient.CoreV1().Secrets(Namespace).Delete(ctx, secretName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete spoke credential secret %s/%s: %w", Namespace, secretName, err)
	}

	return nil
}

func (r *Reconciler) persistHubCredentials(
	ctx context.Context,
	secret *corev1.Secret,
	desiredLabels map[string]string,
	clusterName, server string,
	kubeconfig []byte,
	token, caData string,
	insecure bool,
	issuedAt, expiresAt int64,
) (*corev1.Secret, error) {
	updated := secret.DeepCopy()
	updated.Labels = maps.Clone(desiredLabels)
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	updated.Annotations[IssuedAtAnnotation] = strconv.FormatInt(issuedAt, 10)
	updated.Annotations[ExpiresAtAnnotation] = strconv.FormatInt(expiresAt, 10)
	updated.Annotations[ManagedByAnnotation] = ManagedByValue

	updated.Data = map[string][]byte{
		"name":        []byte(clusterName),
		"server":      []byte(server),
		KubeconfigKey: kubeconfig,
		"token":       []byte(token),
		"caData":      []byte(caData),
		"insecure":    []byte(strconv.FormatBool(insecure)),
	}

	saved, err := r.hubClient.CoreV1().Secrets(Namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("persist credentials in secret %q: %w", secret.Name, err)
	}
	return saved, nil
}

func (r *Reconciler) markSecretReady(ctx context.Context, secret *corev1.Secret, rotatedAt int64) error {
	updated := secret.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	updated.Annotations[RotationStatusAnnotation] = RotationStatusReady
	updated.Annotations[RotatedAtAnnotation] = strconv.FormatInt(rotatedAt, 10)
	delete(updated.Annotations, RotationErrorAnnotation)

	if _, err := r.hubClient.CoreV1().Secrets(Namespace).Update(ctx, updated, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("set ready rotation status on secret %q: %w", secret.Name, err)
	}
	return nil
}

func (r *Reconciler) recordRotationFailure(ctx context.Context, secretName, reason string) {
	secrets := r.hubClient.CoreV1().Secrets(Namespace)
	secret, err := secrets.Get(ctx, secretName, metav1.GetOptions{})
	if err != nil || string(secret.Type) != SecretType {
		return
	}

	updated := secret.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	updated.Annotations[RotationStatusAnnotation] = RotationStatusFailed
	updated.Annotations[RotatedAtAnnotation] = strconv.FormatInt(r.cfg.Now().Unix(), 10)
	updated.Annotations[RotationErrorAnnotation] = reason

	_, _ = secrets.Update(ctx, updated, metav1.UpdateOptions{})
}

type argoClusterConfig struct {
	BearerToken     string        `json:"bearerToken"`
	TLSClientConfig argoTLSConfig `json:"tlsClientConfig"`
}

type argoTLSConfig struct {
	CAData   string `json:"caData"`
	Insecure bool   `json:"insecure"`
}

func (r *Reconciler) projectArgoTarget(
	ctx context.Context,
	clusterName, server, token, caData string,
	insecure bool,
	sourceLabels map[string]string,
	issuedAt, expiresAt int64,
) error {
	cfgJSON, err := json.Marshal(argoClusterConfig{
		BearerToken: token,
		TLSClientConfig: argoTLSConfig{
			CAData:   caData,
			Insecure: insecure,
		},
	})
	if err != nil {
		return fmt.Errorf("marshal Argo CD cluster config: %w", err)
	}

	labels := maps.Clone(sourceLabels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels[ArgoSecretTypeLabel] = ArgoSecretTypeCluster

	data := map[string][]byte{
		"name":        []byte(clusterName),
		"server":      []byte(server),
		ArgoConfigKey: cfgJSON,
	}

	return r.upsertManagedTargetSecret(ctx, r.cfg.ArgoNamespace, SecretNameForCluster(clusterName), labels, data, issuedAt, expiresAt)
}

func (r *Reconciler) projectFluxTarget(
	ctx context.Context,
	clusterName string,
	kubeconfig []byte,
	sourceLabels map[string]string,
	issuedAt, expiresAt int64,
) error {
	labels := maps.Clone(sourceLabels)
	if labels == nil {
		labels = map[string]string{}
	}

	data := map[string][]byte{
		FluxKubeconfigKey: kubeconfig,
	}

	return r.upsertManagedTargetSecret(ctx, r.cfg.FluxNamespace, clusterName+FluxSecretSuffix, labels, data, issuedAt, expiresAt)
}

func (r *Reconciler) upsertManagedTargetSecret(
	ctx context.Context,
	namespace, secretName string,
	labels map[string]string,
	data map[string][]byte,
	issuedAt, expiresAt int64,
) error {
	secrets := r.hubClient.CoreV1().Secrets(namespace)
	annotations := map[string]string{
		IssuedAtAnnotation:  strconv.FormatInt(issuedAt, 10),
		ExpiresAtAnnotation: strconv.FormatInt(expiresAt, 10),
		ManagedByAnnotation: ManagedByValue,
	}

	existing, err := secrets.Get(ctx, secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = secrets.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        secretName,
				Namespace:   namespace,
				Labels:      labels,
				Annotations: annotations,
			},
			Type: corev1.SecretTypeOpaque,
			Data: data,
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create target secret %s/%s: %w", namespace, secretName, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get target secret %s/%s: %w", namespace, secretName, err)
	}

	if existing.Annotations[ManagedByAnnotation] != ManagedByValue {
		return fmt.Errorf(
			"refusing to overwrite unmanaged target secret %s/%s (managed-by: %q)",
			namespace,
			secretName,
			existing.Annotations[ManagedByAnnotation],
		)
	}

	updated := existing.DeepCopy()
	updated.Labels = labels
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	maps.Copy(updated.Annotations, annotations)
	if updated.Data == nil {
		updated.Data = map[string][]byte{}
	}
	maps.Copy(updated.Data, data)

	if _, err := secrets.Update(ctx, updated, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update target secret %s/%s: %w", namespace, secretName, err)
	}

	return nil
}

func (r *Reconciler) deleteManagedTargetSecret(ctx context.Context, namespace, secretName string) error {
	secrets := r.hubClient.CoreV1().Secrets(namespace)
	existing, err := secrets.Get(ctx, secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get target secret %s/%s: %w", namespace, secretName, err)
	}

	if existing.Annotations[ManagedByAnnotation] != ManagedByValue {
		log.Info().
			Str("namespace", namespace).
			Str("secret", secretName).
			Msg("Skipping deletion of unmanaged target secret")
		return nil
	}

	if err := secrets.Delete(ctx, secretName, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete target secret %s/%s: %w", namespace, secretName, err)
	}
	return nil
}

func (r *Reconciler) validateSpokeKubeconfig(ctx context.Context, kubeconfig []byte) error {
	spokeClient, err := r.cfg.SpokeClientFor(kubeconfig)
	if err != nil {
		return err
	}
	_, err = spokeClient.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	return err
}

func ensureSpokeAgent(ctx context.Context, spokeClient kubernetes.Interface, clusterRoleName string) error {
	if _, err := spokeClient.CoreV1().Namespaces().Get(ctx, Namespace, metav1.GetOptions{}); apierrors.IsNotFound(err) {
		if _, err := spokeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: Namespace},
		}, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create namespace %q: %w", Namespace, err)
		}
	} else if err != nil {
		return fmt.Errorf("get namespace %q: %w", Namespace, err)
	}

	if _, err := spokeClient.CoreV1().ServiceAccounts(Namespace).Get(ctx, AgentServiceAccountName, metav1.GetOptions{}); apierrors.IsNotFound(err) {
		if _, err := spokeClient.CoreV1().ServiceAccounts(Namespace).Create(ctx, &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      AgentServiceAccountName,
				Namespace: Namespace,
			},
		}, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create serviceaccount %q: %w", AgentServiceAccountName, err)
		}
	} else if err != nil {
		return fmt.Errorf("get serviceaccount %q: %w", AgentServiceAccountName, err)
	}

	desiredBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: AgentClusterRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      AgentServiceAccountName,
				Namespace: Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
	}

	existingBinding, err := spokeClient.RbacV1().ClusterRoleBindings().Get(ctx, AgentClusterRoleBindingName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		if _, err := spokeClient.RbacV1().ClusterRoleBindings().Create(ctx, desiredBinding, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("create clusterrolebinding %q: %w", AgentClusterRoleBindingName, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("get clusterrolebinding %q: %w", AgentClusterRoleBindingName, err)
	}

	updated := existingBinding.DeepCopy()
	updated.Subjects = desiredBinding.Subjects
	updated.RoleRef = desiredBinding.RoleRef
	if _, err := spokeClient.RbacV1().ClusterRoleBindings().Update(ctx, updated, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update clusterrolebinding %q: %w", AgentClusterRoleBindingName, err)
	}
	return nil
}

func mintSpokeAgentToken(ctx context.Context, spokeClient kubernetes.Interface, duration time.Duration) (string, int64, error) {
	seconds := int64(duration.Seconds())
	req := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &seconds,
		},
	}

	resp, err := spokeClient.CoreV1().
		ServiceAccounts(Namespace).
		CreateToken(ctx, AgentServiceAccountName, req, metav1.CreateOptions{})
	if err != nil {
		return "", 0, err
	}

	token := strings.TrimSpace(resp.Status.Token)
	if token == "" {
		return "", 0, fmt.Errorf("token request returned an empty token")
	}

	expiresAt := resp.Status.ExpirationTimestamp.Time.Unix()
	if expiresAt <= 0 {
		return "", 0, fmt.Errorf("token request returned an invalid expiration timestamp")
	}

	return token, expiresAt, nil
}

func extractClusterEndpoint(kubeconfig []byte) (server, caData string, insecure bool, err error) {
	rawCfg, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return "", "", false, fmt.Errorf("parse kubeconfig: %w", err)
	}

	var cluster *clientcmdapi.Cluster
	if rawCfg.CurrentContext != "" {
		if ctx, ok := rawCfg.Contexts[rawCfg.CurrentContext]; ok && ctx != nil {
			cluster = rawCfg.Clusters[ctx.Cluster]
		}
	}
	if cluster == nil {
		for _, c := range rawCfg.Clusters {
			if c != nil {
				cluster = c
				break
			}
		}
	}
	if cluster == nil || strings.TrimSpace(cluster.Server) == "" {
		return "", "", false, fmt.Errorf("kubeconfig contains no cluster server")
	}

	if len(cluster.CertificateAuthorityData) > 0 {
		caData = base64.StdEncoding.EncodeToString(cluster.CertificateAuthorityData)
	}

	return strings.TrimSpace(cluster.Server), caData, cluster.InsecureSkipTLSVerify, nil
}

func buildAgentKubeconfig(server, caData string, insecure bool, token string) ([]byte, error) {
	if strings.TrimSpace(server) == "" {
		return nil, fmt.Errorf("server is required")
	}

	var caBytes []byte
	if caData != "" && !insecure {
		var err error
		caBytes, err = base64.StdEncoding.DecodeString(caData)
		if err != nil {
			return nil, fmt.Errorf("decode caData: %w", err)
		}
	}

	cfg := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"managed-cluster": {
				Server:                   server,
				InsecureSkipTLSVerify:    insecure,
				CertificateAuthorityData: caBytes,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"managed-cluster": {
				Cluster:  "managed-cluster",
				AuthInfo: AgentServiceAccountName,
			},
		},
		CurrentContext: "managed-cluster",
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			AgentServiceAccountName: {
				Token: token,
			},
		},
	}

	return clientcmd.Write(cfg)
}
