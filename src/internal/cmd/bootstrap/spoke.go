package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/k8s"
	"github.com/rs/zerolog/log"
)

const (
	spokeCredentialsNamespace = "kubara-system"

	bootstrapKubeconfigKey = "bootstrapKubeconfig"
	generatedKubeconfigKey = "kubeconfig"

	clusterCredentialsSecretType = "kubara.io/cluster-credentials"

	clusterNameLabel = "kubara.io/cluster-name"

	rotationStatusAnnotation = "kubara.io/rotation-status"
	rotationErrorAnnotation  = "kubara.io/rotation-error"
)

// BootstrapSpoke onboards an already-existing spoke cluster through the
// credential rotator running on the Hub.
//
// The Hub is reached with opts.Kubeconfig. opts.InitialKubeconfig is stored
// only as the initial bootstrap credential for the target spoke. The rotator
// uses it to create kubara-agent and mint the first rotating credential.
func BootstrapSpoke(ctx context.Context, opts *Options) error {
	if opts.ClusterConfig == nil {
		return fmt.Errorf("cluster configuration is required")
	}

	if opts.ClusterConfig.Type != config.Spoke {
		return fmt.Errorf("cluster %q is not a spoke cluster", opts.ClusterName)
	}

	if opts.InitialKubeconfig == "" {
		return fmt.Errorf("initial kubeconfig is required for spoke onboarding")
	}

	initialKubeconfig, err := os.ReadFile(opts.InitialKubeconfig)
	if err != nil {
		return fmt.Errorf("read initial kubeconfig: %w", err)
	}

	if _, err := clientcmd.Load(initialKubeconfig); err != nil {
		return fmt.Errorf("parse initial kubeconfig: %w", err)
	}

	if opts.ClusterSecretsProvider == config.ClusterSecretsProviderExternalSecrets {
		if err := prepareSpokeCluster(ctx, opts); err != nil {
			return err
		}

		log.Info().Msgf("Spoke cluster %q credentials managed via external-secrets; initial setup completed", opts.ClusterName)
		return nil
	}

	hubClient, err := k8s.NewClient(k8s.Config{
		KubeconfigPath: opts.Kubeconfig,
		QPS:            50,
		Burst:          100,
		Timeout:        30 * time.Second,
		UserAgent:      "kubara-spoke-bootstrap",
	})
	if err != nil {
		return fmt.Errorf("create hub kubernetes client: %w", err)
	}
	client := hubClient.Clientset

	configMapName := opts.ClusterName + "-config"

	configMap, err := client.CoreV1().
		ConfigMaps(spokeCredentialsNamespace).
		Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf(
			"get cluster configuration ConfigMap %q in namespace %q: %w",
			configMapName,
			spokeCredentialsNamespace,
			err,
		)
	}

	if configMap.Labels[clusterNameLabel] != opts.ClusterName {
		return fmt.Errorf(
			"ConfigMap %q does not contain required label %s=%s",
			configMapName,
			clusterNameLabel,
			opts.ClusterName,
		)
	}

	if _, err := getCredsRotatorCronJob(ctx, client); err != nil {
		return err
	}

	if err := prepareSpokeCluster(ctx, opts); err != nil {
		return err
	}

	secretName := opts.ClusterName + "-cluster-secret"

	if err := upsertInitialSpokeSecret(
		ctx,
		client,
		secretName,
		initialKubeconfig,
	); err != nil {
		return err
	}

	log.Info().Msgf("Waiting for credential rotator to onboard spoke %q...", opts.ClusterName)

	if err := waitForSpokeOnboarding(ctx, client, secretName); err != nil {
		return err
	}

	log.Info().Msgf("Spoke cluster %q onboarded successfully", opts.ClusterName)

	return nil
}

func prepareSpokeCluster(ctx context.Context, opts *Options) error {
	crdsChartPath, err := chartPathForService(opts.Catalog, catalog.BootstrapServiceCRDs)
	if err != nil {
		return err
	}

	spokeClient, err := k8s.NewClient(k8s.Config{
		KubeconfigPath: opts.InitialKubeconfig,
		QPS:            50,
		Burst:          100,
		Timeout:        30 * time.Second,
		UserAgent:      "kubara-spoke-bootstrap",
	})
	if err != nil {
		return fmt.Errorf("create spoke kubernetes client: %w", err)
	}

	bootstrapCharts := []BootstrapChart{
		{
			Name:      catalog.BootstrapServiceCRDs,
			Path:      filepath.Join(opts.PlatformComponents, "helm", crdsChartPath),
			Enabled:   true,
			EnsureCRD: true,
		},
	}

	if err := addHelmRepositories(ctx, bootstrapCharts); err != nil {
		return fmt.Errorf("add helm repositories for spoke: %w", err)
	}

	if err := updateHelmDependencies(ctx, opts, bootstrapCharts); err != nil {
		return fmt.Errorf("update helm repositories for spoke: %w", err)
	}

	if err := applyCRDs(ctx, spokeClient, opts, bootstrapCharts); err != nil {
		return fmt.Errorf("apply CRDs to spoke: %w", err)
	}
	spokeClient.RefreshDiscovery()

	if opts.WithESCSSPath != "" {
		secretManager := NewSecretManager(spokeClient)
		clusterSecStore, err := secretManager.createClusterSecretStore(opts)
		if err != nil {
			return fmt.Errorf("create ClusterSecretStore for spoke: %w", err)
		}
		if clusterSecStore != nil {
			yamlData, err := yaml.Marshal(clusterSecStore)
			if err != nil {
				return fmt.Errorf("marshal ClusterSecretStore for spoke: %w", err)
			}
			k8sOpts := k8s.DefaultApplyOptions()
			k8sOpts.FieldManager = "kubara-spoke-secrets"
			if err := spokeClient.ApplyManifest(ctx, yamlData, k8sOpts); err != nil {
				return fmt.Errorf("applying ClusterSecretStore to spoke: %w", err)
			}
			log.Info().Msg("Applied ClusterSecretStore to spoke cluster")
		}
	}

	return nil
}

func upsertInitialSpokeSecret(
	ctx context.Context,
	client kubernetes.Interface,
	secretName string,
	initialKubeconfig []byte,
) error {
	secrets := client.CoreV1().Secrets(spokeCredentialsNamespace)

	secret, err := secrets.Get(ctx, secretName, metav1.GetOptions{})

	if apierrors.IsNotFound(err) {
		_, err = secrets.Create(
			ctx,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: spokeCredentialsNamespace,
				},
				Type: corev1.SecretType(clusterCredentialsSecretType),
				Data: map[string][]byte{
					bootstrapKubeconfigKey: initialKubeconfig,
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
		bootstrapKubeconfigKey: initialKubeconfig,
	}

	if updated.Annotations != nil {
		delete(updated.Annotations, rotationStatusAnnotation)
		delete(updated.Annotations, rotationErrorAnnotation)
	}

	updated.Type = corev1.SecretType(clusterCredentialsSecretType)

	if _, err := secrets.Update(ctx, updated, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update initial cluster secret %q: %w", secretName, err)
	}

	log.Info().Msgf("Reset initial cluster secret %q for onboarding", secretName)
	return nil
}

func getCredsRotatorCronJob(ctx context.Context, client kubernetes.Interface) (batchv1.CronJob, error) {
	cronJobs, err := client.BatchV1().
		CronJobs(spokeCredentialsNamespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=cluster-credential-rotator",
		})
	if err != nil {
		return batchv1.CronJob{}, fmt.Errorf("discover credential rotator CronJob: %w", err)
	}

	if len(cronJobs.Items) == 1 {
		return cronJobs.Items[0], nil
	}

	if len(cronJobs.Items) > 1 {
		return batchv1.CronJob{}, fmt.Errorf(
			"invalid setup: found multiple credential rotator CronJobs in namespace %q",
			spokeCredentialsNamespace,
		)
	}

	return batchv1.CronJob{}, fmt.Errorf(
		"credential rotator CronJob not found in namespace %q; ensure the Hub has reconciled the credential rotator",
		spokeCredentialsNamespace,
	)
}

func waitForSpokeOnboarding(ctx context.Context, client kubernetes.Interface, secretName string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		secret, err := client.CoreV1().
			Secrets(spokeCredentialsNamespace).
			Get(ctx, secretName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("read cluster secret %q: %w", secretName, err)
		}

		switch secret.Annotations[rotationStatusAnnotation] {
		case "ready":
			if len(secret.Data[generatedKubeconfigKey]) == 0 {
				return fmt.Errorf(
					"credential rotator marked secret %q ready but it contains no generated credentials",
					secretName,
				)
			}
			if len(secret.Data["server"]) == 0 {
				return fmt.Errorf(
					"credential rotator marked secret %q ready but it contains no cluster server",
					secretName,
				)
			}
			if len(secret.Data["name"]) == 0 {
				return fmt.Errorf(
					"credential rotator marked secret %q ready but it contains no cluster name",
					secretName,
				)
			}
			return nil

		case "failed":
			reason := strings.TrimSpace(secret.Annotations[rotationErrorAnnotation])
			if reason != "" {
				return fmt.Errorf(
					"credential rotation failed for secret %q: %s (check cluster-credential-rotator logs in namespace %q)",
					secretName,
					reason,
					spokeCredentialsNamespace,
				)
			}
			return fmt.Errorf(
				"credential rotation failed for secret %q (check cluster-credential-rotator logs in namespace %q)",
				secretName,
				spokeCredentialsNamespace,
			)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"waiting for credential rotator to onboard secret %q: %w (check cluster-credential-rotator logs in namespace %q)",
				secretName,
				ctx.Err(),
				spokeCredentialsNamespace,
			)
		case <-ticker.C:
		}
	}
}
