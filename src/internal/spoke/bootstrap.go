package spoke

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func UpsertBootstrapSecret(
	ctx context.Context,
	client kubernetes.Interface,
	clusterName string,
	initialKubeconfig []byte,
) error {
	secretName := SecretNameForCluster(clusterName)
	secrets := client.CoreV1().Secrets(Namespace)

	secret, err := secrets.Get(ctx, secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = secrets.Create(
			ctx,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: Namespace,
				},
				Type: corev1.SecretType(SecretType),
				Data: map[string][]byte{
					BootstrapKubeconfigKey: initialKubeconfig,
				},
			},
			metav1.CreateOptions{},
		)
		if err != nil {
			return fmt.Errorf("create initial cluster secret %q: %w", secretName, err)
		}

		log.Info().Msgf("Created initial cluster secret %q", secretName)
		return nil
	}

	if err != nil {
		return fmt.Errorf("read initial cluster secret %q: %w", secretName, err)
	}

	updated := secret.DeepCopy()
	updated.Data = map[string][]byte{
		BootstrapKubeconfigKey: initialKubeconfig,
	}

	if updated.Annotations != nil {
		delete(updated.Annotations, RotationStatusAnnotation)
		delete(updated.Annotations, RotationErrorAnnotation)
	}

	updated.Type = corev1.SecretType(SecretType)

	if _, err := secrets.Update(ctx, updated, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update initial cluster secret %q: %w", secretName, err)
	}

	log.Info().Msgf("Reset initial cluster secret %q for onboarding", secretName)
	return nil
}

func WaitForOnboarding(ctx context.Context, client kubernetes.Interface, clusterName string) error {
	secretName := SecretNameForCluster(clusterName)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		secret, err := client.CoreV1().
			Secrets(Namespace).
			Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("read cluster secret %q: %w", secretName, err)
		}

		switch secret.Annotations[RotationStatusAnnotation] {
		case RotationStatusReady:
			if len(secret.Data[KubeconfigKey]) == 0 {
				return fmt.Errorf("credential rotator marked secret %q ready but it contains no generated credentials", secretName)
			}
			if len(secret.Data["server"]) == 0 {
				return fmt.Errorf("credential rotator marked secret %q ready but it contains no cluster server", secretName)
			}
			if len(secret.Data["name"]) == 0 {
				return fmt.Errorf("credential rotator marked secret %q ready but it contains no cluster name", secretName)
			}
			return nil

		case RotationStatusFailed:
			reason := strings.TrimSpace(secret.Annotations[RotationErrorAnnotation])
			if reason != "" {
				return fmt.Errorf(
					"credential rotation failed for secret %q: %s (check cluster-credential-rotator logs in namespace %q)",
					secretName,
					reason,
					Namespace,
				)
			}
			return fmt.Errorf(
				"credential rotation failed for secret %q (check cluster-credential-rotator logs in namespace %q)",
				secretName,
				Namespace,
			)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"waiting for credential rotator to onboard secret %q: %w (check cluster-credential-rotator logs in namespace %q)",
				secretName,
				ctx.Err(),
				Namespace,
			)
		case <-ticker.C:
		}
	}
}
