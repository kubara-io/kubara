package spoke

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const testBootstrapKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: spoke-a
  cluster:
    server: https://spoke-a.example.com:6443
    insecure-skip-tls-verify: true
contexts:
- name: spoke-a
  context:
    cluster: spoke-a
    user: admin
current-context: spoke-a
users:
- name: admin
  user:
    token: initial-admin-token
`

func TestReconciler_OnboardsSpokeAndProjectsArgoSecret(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1700000000, 0)

	hubClient := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spoke-a-config",
				Namespace: Namespace,
				Labels: map[string]string{
					ClusterNameLabel: "spoke-a",
					"stage":          "dev",
				},
			},
		},
	)

	spokeClient := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}},
	)
	spokeClient.PrependReactor("create", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.GetSubresource() != "token" {
			return false, nil, nil
		}
		return true, &authenticationv1.TokenRequest{
			Status: authenticationv1.TokenRequestStatus{
				Token:               "minted-spoke-token",
				ExpirationTimestamp: metav1.NewTime(now.Add(time.Hour)),
			},
		}, nil
	})

	require.NoError(t, UpsertBootstrapSecret(ctx, hubClient, "spoke-a", []byte(testBootstrapKubeconfig)))

	reconciler, err := NewReconciler(hubClient, Config{
		TokenDuration: time.Hour,
		RefreshBefore: 15 * time.Minute,
		ArgoEnabled:   true,
		Now:           func() time.Time { return now },
		SpokeClientFor: func([]byte) (kubernetes.Interface, error) {
			return spokeClient, nil
		},
	})
	require.NoError(t, err)
	require.NoError(t, reconciler.ReconcileAll(ctx))
	require.NoError(t, WaitForOnboarding(ctx, hubClient, "spoke-a"))

	hubSecret, err := hubClient.CoreV1().Secrets(Namespace).Get(ctx, "spoke-a-cluster-secret", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Empty(t, hubSecret.Data[BootstrapKubeconfigKey])
	assert.Equal(t, "minted-spoke-token", string(hubSecret.Data["token"]))
	assert.Equal(t, RotationStatusReady, hubSecret.Annotations[RotationStatusAnnotation])

	argoSecret, err := hubClient.CoreV1().Secrets("argocd").Get(ctx, "spoke-a-cluster-secret", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, ArgoSecretTypeCluster, argoSecret.Labels[ArgoSecretTypeLabel])
	assert.Equal(t, "dev", argoSecret.Labels["stage"])
	assert.Contains(t, string(argoSecret.Data[ArgoConfigKey]), "minted-spoke-token")
}

func TestReconciler_DecommissionsSpokeWhenConfigMapRemoved(t *testing.T) {
	ctx := context.Background()
	hubClient := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spoke-a-cluster-secret",
				Namespace: Namespace,
			},
			Type: corev1.SecretType(SecretType),
			Data: map[string][]byte{"name": []byte("spoke-a")},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spoke-a-cluster-secret",
				Namespace: "argocd",
				Annotations: map[string]string{
					ManagedByAnnotation: ManagedByValue,
				},
			},
			Type: corev1.SecretTypeOpaque,
		},
	)

	reconciler, err := NewReconciler(hubClient, Config{
		TokenDuration: time.Hour,
		RefreshBefore: 15 * time.Minute,
		ArgoEnabled:   true,
	})
	require.NoError(t, err)
	require.NoError(t, reconciler.ReconcileAll(ctx))

	_, err = hubClient.CoreV1().Secrets(Namespace).Get(ctx, "spoke-a-cluster-secret", metav1.GetOptions{})
	assert.Error(t, err)
	_, err = hubClient.CoreV1().Secrets("argocd").Get(ctx, "spoke-a-cluster-secret", metav1.GetOptions{})
	assert.Error(t, err)
}
