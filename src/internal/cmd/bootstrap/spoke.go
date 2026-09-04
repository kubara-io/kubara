package bootstrap

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubara-io/kubara/internal/config"
	"github.com/rs/zerolog/log"
)

const (
	spokeCredentialsNamespace = "argocd"

	agentNamespaceAnnotation = "kubara.io/agent-namespace"

	bootstrapKubeconfigKey = "bootstrapKubeconfig"
	generatedConfigKey     = "config"

	clusterCredentialsSecretType = "kubara.io/cluster-credentials"

	argoClusterLabel      = "argocd.argoproj.io/secret-type"
	argoClusterLabelValue = "cluster"

	targetSecretEnvName = "KUBARA_TARGET_SECRET_NAME"
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

	agentNamespace := opts.AgentNamespace
	if agentNamespace == "" {
		agentNamespace = "kubara"
	}

	if errs := validation.IsDNS1123Label(agentNamespace); len(errs) > 0 {
		return fmt.Errorf("invalid kubara-agent namespace %q: %s", agentNamespace, strings.Join(errs, "; "))
	}

	initialKubeconfig, err := os.ReadFile(opts.InitialKubeconfig)
	if err != nil {
		return fmt.Errorf("read initial kubeconfig: %w", err)
	}

	if _, err := clientcmd.Load(initialKubeconfig); err != nil {
		return fmt.Errorf("parse initial kubeconfig: %w", err)
	}

	restConfig, err := clientcmd.BuildConfigFromFlags("", opts.Kubeconfig)
	if err != nil {
		return fmt.Errorf("load hub kubeconfig: %w", err)
	}

	restConfig.QPS = 50
	restConfig.Burst = 100
	restConfig.Timeout = 30 * time.Second
	restConfig.UserAgent = "kubara-spoke-bootstrap"

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create hub kubernetes client: %w", err)
	}

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

	if configMap.Labels[argoClusterLabel] != argoClusterLabelValue {
		return fmt.Errorf(
			"ConfigMap %q does not contain required label %s=%s",
			configMapName,
			argoClusterLabel,
			argoClusterLabelValue,
		)
	}

	cronJob, err := getCredsRotatorCronJob(ctx, client)
	if err != nil {
		return err
	}

	secretName := opts.ClusterName + "-cluster-secret"

	if err := upsertInitialSpokeSecret(
		ctx,
		client,
		secretName,
		initialKubeconfig,
		agentNamespace,
	); err != nil {
		return err
	}

	job, err := startOnboardingJob(ctx, client, cronJob, secretName)
	if err != nil {
		return err
	}

	log.Info().Msgf("Started credential rotator job %q for spoke %q", job.Name, opts.ClusterName)

	if err := waitForRotatorJob(ctx, client, job.Name); err != nil {
		logCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if jobLogs := getRotatorJobLogs(logCtx, client, job.Name); jobLogs != "" {
			log.Error().Msgf("credential rotator logs:\n%s", jobLogs)
		}

		return err
	}

	secret, err := client.CoreV1().
		Secrets(spokeCredentialsNamespace).
		Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("read onboarded cluster secret %q: %w", secretName, err)
	}

	if len(secret.Data[generatedConfigKey]) == 0 {
		return fmt.Errorf(
			"credential rotator job completed but secret %q contains no generated credentials",
			secretName,
		)
	}

	if len(secret.Data["server"]) == 0 {
		return fmt.Errorf("credential rotator job completed but secret %q contains no cluster server", secretName)
	}

	if len(secret.Data["name"]) == 0 {
		return fmt.Errorf("credential rotator job completed but secret %q contains no cluster name", secretName)
	}

	if secret.Annotations[agentNamespaceAnnotation] != agentNamespace {
		return fmt.Errorf(
			"credential rotator job completed but secret %q has unexpected %s annotation %q",
			secretName,
			agentNamespaceAnnotation,
			secret.Annotations[agentNamespaceAnnotation],
		)
	}

	if !reflect.DeepEqual(secret.Labels, configMap.Labels) {
		return fmt.Errorf(
			"credential rotator job completed but labels on secret %q do not exactly match authoritative ConfigMap %q",
			secretName,
			configMapName,
		)
	}

	log.Info().Msgf(
		"Spoke cluster %q onboarded successfully with kubara-agent namespace %q",
		opts.ClusterName,
		agentNamespace,
	)

	return nil
}

func upsertInitialSpokeSecret(
	ctx context.Context,
	client kubernetes.Interface,
	secretName string,
	initialKubeconfig []byte,
	agentNamespace string,
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
					Annotations: map[string]string{
						agentNamespaceAnnotation: agentNamespace,
					},
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

	if len(secret.Data[generatedConfigKey]) > 0 {
		return fmt.Errorf(
			"cluster secret %q already contains generated credentials; refusing to re-onboard it",
			secretName,
		)
	}

	updated := secret.DeepCopy()

	if updated.Data == nil {
		updated.Data = map[string][]byte{}
	}
	updated.Data[bootstrapKubeconfigKey] = initialKubeconfig

	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	updated.Annotations[agentNamespaceAnnotation] = agentNamespace

	updated.Type = corev1.SecretType(clusterCredentialsSecretType)

	if _, err := secrets.Update(ctx, updated, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update initial cluster secret %q: %w", secretName, err)
	}

	log.Info().Msgf("Updated initial cluster secret %q", secretName)
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

func startOnboardingJob(
	ctx context.Context,
	client kubernetes.Interface,
	cronJob batchv1.CronJob,
	secretName string,
) (*batchv1.Job, error) {
	jobTemplate := cronJob.Spec.JobTemplate.DeepCopy()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cronJob.Name + "-bootstrap-",
			Namespace:    spokeCredentialsNamespace,
			Labels:       jobTemplate.Labels,
			Annotations:  jobTemplate.Annotations,
		},
		Spec: jobTemplate.Spec,
	}

	if job.Labels == nil {
		job.Labels = map[string]string{}
	}
	job.Labels["app.kubernetes.io/name"] = "cluster-credential-rotator"
	job.Labels["app.kubernetes.io/managed-by"] = "kubara"

	if job.Annotations == nil {
		job.Annotations = map[string]string{}
	}
	job.Annotations["cronjob.kubernetes.io/instantiate"] = "manual"

	if len(job.Spec.Template.Spec.Containers) != 1 {
		return nil, fmt.Errorf(
			"credential rotator CronJob must contain exactly one container, found %d",
			len(job.Spec.Template.Spec.Containers),
		)
	}

	container := &job.Spec.Template.Spec.Containers[0]
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  targetSecretEnvName,
			Value: secretName,
		},
	)

	createdJob, err := client.BatchV1().
		Jobs(spokeCredentialsNamespace).
		Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create credential rotator Job: %w", err)
	}

	return createdJob, nil
}

func waitForRotatorJob(ctx context.Context, client kubernetes.Interface, jobName string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		job, err := client.BatchV1().
			Jobs(spokeCredentialsNamespace).
			Get(ctx, jobName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("read credential rotator Job %q: %w", jobName, err)
		}

		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobComplete &&
				condition.Status == corev1.ConditionTrue {
				return nil
			}

			if condition.Type == batchv1.JobFailed &&
				condition.Status == corev1.ConditionTrue {
				message := condition.Message
				if message == "" {
					message = condition.Reason
				}
				if message == "" {
					message = "Job reported failed condition"
				}

				return fmt.Errorf("credential rotator Job %q failed: %s", jobName, message)
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for credential rotator Job %q: %w", jobName, ctx.Err())
		case <-ticker.C:
		}
	}
}

func getRotatorJobLogs(ctx context.Context, client kubernetes.Interface, jobName string) string {
	pods, err := client.CoreV1().
		Pods(spokeCredentialsNamespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: "job-name=" + jobName,
		})
	if err != nil {
		return ""
	}

	var output []string

	for _, pod := range pods.Items {
		data, err := client.CoreV1().
			Pods(spokeCredentialsNamespace).
			GetLogs(pod.Name, &corev1.PodLogOptions{}).
			DoRaw(ctx)
		if err != nil {
			continue
		}

		logs := strings.TrimSpace(string(data))
		if logs == "" {
			continue
		}

		output = append(output, fmt.Sprintf("[%s]\n%s", pod.Name, logs))
	}

	return strings.Join(output, "\n")
}
