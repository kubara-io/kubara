package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kubara-io/kubara/internal/catalog"
	generatecmd "github.com/kubara-io/kubara/internal/cmd/generate"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/helm"
	"github.com/kubara-io/kubara/internal/k8s"
	"github.com/kubara-io/kubara/internal/localmode"
	"github.com/kubara-io/kubara/internal/render"
	"github.com/kubara-io/kubara/internal/utils"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	localDirectoryName          = ".local"
	localKindConfigName         = "kind-config.yaml"
	localKindHostCACertPath     = "/etc/ssl/certs/ca-certificates.crt"
	localTraefikNamespace       = "traefik"
	localTraefikReleaseName     = "traefik"
	localOpenBaoNamespace       = "openbao"
	localOpenBaoReleaseName     = "openbao"
	localOpenBaoPodName         = "openbao-0"
	localExternalSecretsRole    = "any-sa"
	localExternalSecretsPolicy  = "read-access"
	localOpenBaoInternalAddress = "http://openbao.openbao.svc:8200"
	// renovate: datasource=helm depName=openbao registryUrl=https://openbao.github.io/openbao-helm
	localOpenBaoChartVersion = "0.29.1"
)

type localChart struct {
	DisplayName string
	Repo        helm.RepoOptions
	ReleaseName string
	ChartRef    string
	Version     string
	Namespace   string
}

var (
	localOpenBaoChart = localChart{"OpenBao", helm.RepoOptions{Name: "openbao", URL: "https://openbao.github.io/openbao-helm"}, localOpenBaoReleaseName, "openbao/openbao", localOpenBaoChartVersion, localOpenBaoNamespace}
)

func prepareLocalBootstrap(ctx context.Context, opts *Options) error {
	if opts.ClusterConfig == nil {
		return fmt.Errorf("cluster config is required for --local")
	}

	log.Info().
		Str("cluster", opts.ClusterName).
		Str("runtimeDir", filepath.Join(opts.WorkDir, localDirectoryName)).
		Msg("Preparing local evaluation bootstrap")

	if err := utils.AddGitignore(opts.WorkDir); err != nil {
		return fmt.Errorf("ensure .gitignore contains kubara local entries: %w", err)
	}

	state := newLocalState(opts)
	opts.LocalState = state

	log.Info().Msg("Checking local bootstrap prerequisites")
	if err := ensureLocalPrerequisites(); err != nil {
		return err
	}
	if err := os.MkdirAll(state.RuntimeDir, 0o750); err != nil {
		return fmt.Errorf("create local runtime directory: %w", err)
	}

	log.Info().
		Str("cluster", opts.ClusterName).
		Str("kubeconfig", state.KubeconfigPath).
		Msg("Ensuring local kind cluster")
	log.Info().
		Str("configFile", state.KindConfigPath).
		Msg("Writing local kind configuration")
	if err := writeLocalKindConfig(state); err != nil {
		return err
	}
	if err := ensureKindCluster(ctx, opts.ClusterName, state.KubeconfigPath, state.KindConfigPath); err != nil {
		return err
	}

	log.Info().Msg("Connecting to the local Kubernetes cluster")
	client, err := k8s.NewClient(k8s.Config{
		KubeconfigPath: state.KubeconfigPath,
		QPS:            50,
		Burst:          100,
		Timeout:        30 * time.Second,
		UserAgent:      "kubara-local-bootstrap",
	})
	if err != nil {
		return fmt.Errorf("create local kubernetes client: %w", err)
	}

	log.Info().Msg("Creating local Traefik bootstrap namespace and placeholder LoadBalancer service")
	if err := ensureLocalTraefikBootstrapService(ctx, client); err != nil {
		return err
	}

	log.Info().Msg("Waiting for the local Traefik LoadBalancer IP")
	log.Info().Msg(`
Now start cloud-provider-kind in another terminal with: sudo cloud-provider-kind

`)
	loadBalancerIP, err := waitForTraefikLoadBalancer(ctx, client)
	if err != nil {
		return err
	}
	state.LoadBalancerIP = loadBalancerIP
	state.BaseHost = fmt.Sprintf("%s.traefik.me", loadBalancerIP)
	state.OpenBaoHost = fmt.Sprintf("openbao.%s", state.BaseHost)
	log.Info().
		Str("loadBalancerIP", state.LoadBalancerIP).
		Str("baseHost", state.BaseHost).
		Str("openBaoHost", state.OpenBaoHost).
		Msg("Local ingress hostnames are ready")

	log.Info().
		Str("configFile", opts.ConfigFilePath).
		Msg("Updating config.yaml for the local evaluation profile")
	if err := updateLocalClusterConfigAndGenerate(opts, state); err != nil {
		return err
	}
	log.Info().
		Str("valuesFile", state.OpenBaoValuesPath).
		Msg("Writing local OpenBao values")
	if err := writeLocalOpenBaoValues(state); err != nil {
		return err
	}
	log.Info().Msg("Installing local OpenBao")
	if err := installLocalChart(ctx, state.KubeconfigPath, localOpenBaoChart, state.OpenBaoValuesPath, opts.Timeout); err != nil {
		return err
	}
	log.Info().Msg("Waiting for OpenBao to be ready for configuration")
	if err := waitForLocalOpenBaoReady(ctx, client, state.KubeconfigPath); err != nil {
		return err
	}

	log.Info().Msg("Configuring OpenBao for local external-secrets access")
	// Intentianolly using the ArgoCD Wizard password for grafana as well. For convenience on the local evaluation
	// environment and not requiring the user to fill out yet another variable during bootstrapping.
	if err := configureLocalOpenBao(ctx, state.KubeconfigPath, opts.ClusterConfig, opts.EnvMap.ArgocdWizardAccountPassword, opts.EnvMap.DockerconfigBase64); err != nil {
		return err
	}
	log.Info().
		Str("clusterSecretStore", state.ClusterSecretStorePath).
		Msg("Writing local ClusterSecretStore manifest")
	if err := writeLocalClusterSecretStore(opts, state); err != nil {
		return err
	}
	log.Info().
		Str("valuesFile", state.ArgocdValuesPath).
		Msg("Writing local Argo CD overrides")
	if err := writeLocalArgocdValues(opts, state); err != nil {
		return err
	}
	log.Info().
		Str("valuesFile", state.CertManagerValuesPath).
		Msg("Writing local cert-manager overrides")
	if err := writeLocalCertManagerValues(state); err != nil {
		return err
	}
	log.Info().Msg("Writing local ingress overrides for services without oauth2-proxy")
	if err := writeLocalIngressOverrides(state); err != nil {
		return err
	}
	if err := removeLocalFileIfExists(state.OAuth2ProxyValuesPath); err != nil {
		return err
	}

	opts.WithESCSSPath = state.ClusterSecretStorePath
	opts.ClusterConfig.DNSName = state.BaseHost

	log.Info().Msg("Local bootstrap preparation completed")
	return nil
}

func newLocalState(opts *Options) *LocalState {
	repoLocalRuntime := filepath.Join(opts.WorkDir, localDirectoryName)

	return &LocalState{
		RuntimeDir:             repoLocalRuntime,
		KubeconfigPath:         filepath.Join(repoLocalRuntime, "kind.kubeconfig"),
		KindConfigPath:         filepath.Join(repoLocalRuntime, localKindConfigName),
		TraefikOverlayPath:     filepath.Join(opts.PlatformConfigs, opts.ClusterName, "helm", "traefik", "values-additional.yaml"),
		CertManagerValuesPath:  filepath.Join(opts.PlatformConfigs, opts.ClusterName, "helm", "cert-manager", "values-additional.yaml"),
		OpenBaoValuesPath:      filepath.Join(repoLocalRuntime, "openbao", "values.yaml"),
		ArgocdValuesPath:       filepath.Join(opts.PlatformConfigs, opts.ClusterName, "helm", "argo-cd", "values-additional.yaml"),
		HomerValuesPath:        filepath.Join(opts.PlatformConfigs, opts.ClusterName, "helm", "homer-dashboard", "values-additional.yaml"),
		PrometheusValuesPath:   filepath.Join(opts.PlatformConfigs, opts.ClusterName, "helm", "kube-prometheus-stack", "values-additional.yaml"),
		KyvernoValuesPath:      filepath.Join(opts.PlatformConfigs, opts.ClusterName, "helm", "kyverno-policy-reporter", "values-additional.yaml"),
		LonghornValuesPath:     filepath.Join(opts.PlatformConfigs, opts.ClusterName, "helm", "longhorn", "values-additional.yaml"),
		OAuth2ProxyValuesPath:  filepath.Join(opts.PlatformConfigs, opts.ClusterName, "helm", "oauth2-proxy", "values-additional.yaml"),
		ClusterSecretStorePath: filepath.Join(repoLocalRuntime, "external-secrets", "clustersecretstore.yaml"),
		GenerateEnvPath:        filepath.Join(repoLocalRuntime, "generate.env"),
	}
}

func ensureLocalPrerequisites() error {
	for _, command := range []string{"kind", "kubectl", "helm", "cloud-provider-kind"} {
		if _, err := exec.LookPath(command); err != nil {
			return fmt.Errorf("%q is required for --local and must be available on PATH", command)
		}
	}
	for _, command := range []string{"podman", "docker"} {
		if _, err := exec.LookPath(command); err == nil {
			return nil
		}
	}
	return fmt.Errorf("either podman or docker is required for --local and must be available on PATH")
}

func ensureKindCluster(ctx context.Context, clusterName, kubeconfigPath, kindConfigPath string) error {
	output, err := runCommand(ctx, "kind", nil, "", "get", "clusters")
	if err != nil {
		return fmt.Errorf("list kind clusters: %w", err)
	}

	clusterExists := slices.Contains(strings.Fields(string(output)), clusterName)

	env := map[string]string{"KUBECONFIG": kubeconfigPath}
	if clusterExists {
		log.Info().Str("cluster", clusterName).Msg("Reusing existing kind cluster")
		log.Warn().
			Str("cluster", clusterName).
			Msg("Existing kind clusters keep their original extraMounts. Recreate the cluster if you need updated kind mount configuration.")
		if _, err := runCommand(ctx, "kind", env, "", "export", "kubeconfig", "--name", clusterName); err != nil {
			return fmt.Errorf("export kubeconfig for kind cluster %q: %w", clusterName, err)
		}
		return nil
	}

	log.Info().Str("cluster", clusterName).Msg("Creating kind cluster")
	if _, err := runCommand(ctx, "kind", env, "", "create", "cluster", "--name", clusterName, "--config", kindConfigPath); err != nil {
		return fmt.Errorf("create kind cluster %q: %w", clusterName, err)
	}
	return nil
}

func writeLocalKindConfig(state *LocalState) error {
	// Does not need to be guarded against. It is guarenteed that any developer system that can run kind is based on linux
	// and therefore will have local ca-certificates installed
	content := fmt.Sprintf(`kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
  - hostPath: %s
    containerPath: %s
    readOnly: true
`, localKindHostCACertPath, localKindHostCACertPath)

	return writeLocalFile(state.KindConfigPath, content)
}

func installLocalChart(ctx context.Context, kubeconfigPath string, chart localChart, valuesPath string, timeout time.Duration) error {
	if err := helm.AddRepository(ctx, chart.Repo); err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("add %s helm repository: %w", chart.DisplayName, err)
	}
	if err := helm.UpdateRepository(ctx, chart.Repo); err != nil {
		return fmt.Errorf("update %s helm repository: %w", chart.DisplayName, err)
	}
	return helmUpgradeInstall(ctx, kubeconfigPath, chart.ReleaseName, chart.ChartRef, chart.Version, chart.Namespace, valuesPath, timeout)
}

func ensureLocalTraefikBootstrapService(ctx context.Context, client *k8s.Client) error {
	if err := client.EnsureNamespace(ctx, localTraefikNamespace, false); err != nil {
		return fmt.Errorf("ensure local Traefik namespace: %w", err)
	}

	_, err := client.Clientset.CoreV1().Services(localTraefikNamespace).Get(ctx, localTraefikReleaseName, metav1.GetOptions{})
	if err == nil {
		log.Info().
			Str("namespace", localTraefikNamespace).
			Str("service", localTraefikReleaseName).
			Msg("Reusing existing local Traefik service")
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get local Traefik service: %w", err)
	}

	service := buildLocalTraefikBootstrapService()
	if _, err := client.Clientset.CoreV1().Services(localTraefikNamespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("create local Traefik bootstrap service: %w", err)
	}

	return nil
}

func buildLocalTraefikBootstrapService() *corev1.Service {
	instanceName := fmt.Sprintf("traefik-%s", localTraefikReleaseName)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      localTraefikReleaseName,
			Namespace: localTraefikNamespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "traefik",
				"app.kubernetes.io/instance":   instanceName,
				"app.kubernetes.io/managed-by": "Helm",
			},
			Annotations: map[string]string{
				"meta.helm.sh/release-name":      localTraefikReleaseName,
				"meta.helm.sh/release-namespace": localTraefikNamespace,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{
				"app.kubernetes.io/name":     "traefik",
				"app.kubernetes.io/instance": instanceName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       80,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("web"),
				},
				{
					Name:       "websecure",
					Port:       443,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("websecure"),
				},
			},
		},
	}
}

func waitForLocalOpenBaoReady(ctx context.Context, client *k8s.Client, kubeconfigPath string) error {
	if err := client.WaitForPod(ctx, localOpenBaoNamespace, fmt.Sprintf("statefulset.kubernetes.io/pod-name=%s", localOpenBaoPodName)); err != nil {
		return fmt.Errorf("wait for OpenBao pod to be ready: %w", err)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastErr error

	for {
		// Run 'bao status' directly via runCommand (no token needed for status checks)
		_, err := runCommand(ctx, "kubectl", map[string]string{"KUBECONFIG": kubeconfigPath}, "",
			"-n", localOpenBaoNamespace,
			"exec", localOpenBaoPodName,
			"-c", "openbao",
			"--",
			"bao", "status",
		)
		if err == nil {
			log.Info().Msg("OpenBao is ready for configuration")
			return nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("wait for OpenBao command readiness after last error %v: %w", lastErr, ctx.Err())
			}
			return fmt.Errorf("wait for OpenBao command readiness: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func helmUpgradeInstall(ctx context.Context, kubeconfigPath, releaseName, chartRef, chartVersion, namespace, valuesPath string, timeout time.Duration) error {
	args := []string{
		"upgrade",
		"--install",
		releaseName,
		chartRef,
		"--version",
		chartVersion,
		"--namespace",
		namespace,
		"--create-namespace",
		"--wait",
		"--kubeconfig",
		kubeconfigPath,
	}
	if valuesPath != "" {
		args = append(args, "--values", valuesPath)
	}
	if timeout > 0 {
		args = append(args, "--timeout", timeout.String())
	}
	if _, err := runCommand(ctx, "helm", nil, "", args...); err != nil {
		return fmt.Errorf("helm upgrade --install %q: %w", releaseName, err)
	}
	return nil
}

func waitForTraefikLoadBalancer(ctx context.Context, client *k8s.Client) (string, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		svc, err := client.Clientset.CoreV1().Services(localTraefikNamespace).Get(ctx, localTraefikReleaseName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				goto wait
			}
			return "", fmt.Errorf("get Traefik service: %w", err)
		}
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if strings.TrimSpace(ingress.IP) != "" {
				log.Info().
					Str("service", svc.Name).
					Str("ip", ingress.IP).
					Msg("Traefik LoadBalancer IP assigned")
				return ingress.IP, nil
			}
		}

	wait:
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("wait for Traefik LoadBalancer IP: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func updateLocalClusterConfigAndGenerate(opts *Options, state *LocalState) error {
	configStore := config.NewConfigStore(opts.WorkDir, opts.ConfigFilePath, catalog.LoadOptions{
		Catalogs:  opts.Catalogs,
		Overwrite: opts.CatalogOverwrite,
	})
	if err := configStore.Load(); err != nil {
		return fmt.Errorf("load config for local profile update: %w", err)
	}

	var clusterConfig *config.Cluster
	for i := range configStore.GetConfig().Clusters {
		if configStore.GetConfig().Clusters[i].Name == opts.ClusterName {
			clusterConfig = &configStore.GetConfig().Clusters[i]
			break
		}
	}
	if clusterConfig == nil {
		return fmt.Errorf("cluster %q not found in config file %q", opts.ClusterName, opts.ConfigFilePath)
	}

	localmode.ApplyClusterProfile(clusterConfig, state.BaseHost)
	if err := configStore.SaveToFile(); err != nil {
		return fmt.Errorf("save config with local profile: %w", err)
	}
	opts.ClusterConfig = clusterConfig

	localEnvPath, err := writeLocalGenerateEnvFile(state, opts.EnvMap)
	if err != nil {
		return err
	}

	log.Info().
		Str("envFile", localEnvPath).
		Msg("Regenerating local Helm artifacts")
	generateOptions := &generatecmd.Options{
		TemplateType:       render.Helm,
		DryRun:             false,
		CWD:                opts.WorkDir,
		ConfigFilePath:     opts.ConfigFilePath,
		Catalogs:           opts.Catalogs,
		CatalogOverwrite:   opts.CatalogOverwrite,
		PlatformComponents: opts.PlatformComponents,
		PlatformConfigs:    opts.PlatformConfigs,
		EnvPath:            localEnvPath,
	}
	if err := generateOptions.Run(); err != nil {
		return fmt.Errorf("generate Helm catalog for local profile: %w", err)
	}
	return nil
}

func writeLocalGenerateEnvFile(state *LocalState, envMap *envconfig.EnvMap) (string, error) {
	envPath := state.GenerateEnvPath
	content, err := envconfig.RenderEnvFileFromValues(envMap)
	if err != nil {
		return "", fmt.Errorf("render local env file for generation: %w", err)
	}
	if err := writeLocalFile(envPath, string(content)); err != nil {
		return "", fmt.Errorf("write local env file for generation: %w", err)
	}
	return envPath, nil
}

func configureLocalOpenBao(ctx context.Context, kubeconfigPath string, cluster *config.Cluster, grafanaAdminPassword, dockerconfigBase64 string) error {
	if err := ensureOpenBaoSecretEngine(ctx, kubeconfigPath); err != nil {
		return err
	}
	if err := ensureOpenBaoAuthMethod(ctx, kubeconfigPath, "kubernetes"); err != nil {
		return err
	}

	if _, err := kubectlExec(ctx, kubeconfigPath,
		"-n", localOpenBaoNamespace,
		"exec", localOpenBaoPodName,
		"--",
		"bao", "write", "auth/kubernetes/config",
		"kubernetes_host=https://kubernetes.default.svc:443",
	); err != nil {
		return fmt.Errorf("configure OpenBao kubernetes auth: %w", err)
	}

	policyDocument := `path "kv/data/*" {
  capabilities = ["read", "list"]
}

path "kv/metadata/*" {
  capabilities = ["read", "list"]
}
`
	if _, err := kubectlExecWithInput(ctx, kubeconfigPath, policyDocument,
		"-n", localOpenBaoNamespace,
		"exec", "-i", localOpenBaoPodName,
		"--",
		"bao", "policy", "write", localExternalSecretsPolicy, "-",
	); err != nil {
		return fmt.Errorf("configure OpenBao policy: %w", err)
	}

	// The local evaluation role is intentionally wide open so generated services
	// can use external-secrets without per-service role configuration. This does
	// not reflect a production security model or best practices.
	if _, err := kubectlExec(ctx, kubeconfigPath,
		"-n", localOpenBaoNamespace,
		"exec", localOpenBaoPodName,
		"--",
		"bao", "write",
		fmt.Sprintf("auth/kubernetes/role/%s", localExternalSecretsRole),
		"bound_service_account_names=*",
		"bound_service_account_namespaces=*",
		fmt.Sprintf("policies=%s", localExternalSecretsPolicy),
		"ttl=24h",
	); err != nil {
		return fmt.Errorf("configure OpenBao kubernetes role: %w", err)
	}

	secretPathPrefix := fmt.Sprintf("%s/%s", cluster.Name, cluster.Stage)
	if err := writeOpenBaoKVSecret(ctx, kubeconfigPath, secretPathPrefix+"/kube-prometheus-stack/grafana_credentials", map[string]string{
		"admin-user":     "wizard",
		"admin-password": grafanaAdminPassword,
	}); err != nil {
		return err
	}

	if envconfig.IsConfiguredEnvValue(dockerconfigBase64) {
		dockerconfig, err := utils.DecodeB64(dockerconfigBase64)
		if err != nil {
			return fmt.Errorf("decode DOCKERCONFIG_BASE64 for local OpenBao: %w", err)
		}
		if err := writeOpenBaoKVSecret(ctx, kubeconfigPath, secretPathPrefix+"/cluster_secrets/docker_config", map[string]string{
			"pull-secret": dockerconfig,
		}); err != nil {
			return err
		}
	}

	return nil
}

func ensureOpenBaoSecretEngine(ctx context.Context, kubeconfigPath string) error {
	return ensureOpenBaoMount(ctx, kubeconfigPath, openBaoMount{
		Name:              "kv",
		ListDescription:   "OpenBao secrets engines",
		EnableDescription: "OpenBao kv secrets engine",
		ListArgs:          []string{"secrets", "list", "-format=json"},
		EnableArgs:        []string{"secrets", "enable", "-path=kv", "kv-v2"},
	})
}

func ensureOpenBaoAuthMethod(ctx context.Context, kubeconfigPath, method string) error {
	return ensureOpenBaoMount(ctx, kubeconfigPath, openBaoMount{
		Name:              method,
		ListDescription:   "OpenBao auth methods",
		EnableDescription: fmt.Sprintf("OpenBao auth method %q", method),
		ListArgs:          []string{"auth", "list", "-format=json"},
		EnableArgs:        []string{"auth", "enable", method},
	})
}

type openBaoMount struct {
	Name              string
	ListDescription   string
	EnableDescription string
	ListArgs          []string
	EnableArgs        []string
}

func ensureOpenBaoMount(ctx context.Context, kubeconfigPath string, mount openBaoMount) error {
	raw, err := kubectlBao(ctx, kubeconfigPath, mount.ListArgs...)
	if err != nil {
		return fmt.Errorf("list %s: %w", mount.ListDescription, err)
	}

	var mounts map[string]any
	if err := json.Unmarshal(raw, &mounts); err != nil {
		return fmt.Errorf("decode %s: %w", mount.ListDescription, err)
	}
	if _, exists := mounts[mount.Name+"/"]; exists {
		return nil
	}

	if _, err := kubectlBao(ctx, kubeconfigPath, mount.EnableArgs...); err != nil {
		return fmt.Errorf("enable %s: %w", mount.EnableDescription, err)
	}
	return nil
}

func kubectlBao(ctx context.Context, kubeconfigPath string, args ...string) ([]byte, error) {
	kubectlArgs := []string{
		"-n", localOpenBaoNamespace,
		"exec", localOpenBaoPodName,
		"--",
		"bao",
	}
	kubectlArgs = append(kubectlArgs, args...)
	return kubectlExec(ctx, kubeconfigPath, kubectlArgs...)
}

func kubectlExec(ctx context.Context, kubeconfigPath string, args ...string) ([]byte, error) {
	return kubectlExecWithInput(ctx, kubeconfigPath, "", args...)
}

func kubectlExecWithInput(ctx context.Context, kubeconfigPath, input string, args ...string) ([]byte, error) {
	token, err := getOpenBaoRootToken(ctx, kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("get local openbao token: %w", err)
	}

	var targetCmd []string
	foundDashDash := false

	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			foundDashDash = true
			targetCmd = args[i+1:]
			break
		}
	}

	if foundDashDash && len(targetCmd) > 0 {
		execArgs := localOpenBaoExecArgs(token, input != "", targetCmd...)
		out, err := runCommand(ctx, "kubectl", map[string]string{"KUBECONFIG": kubeconfigPath}, input, execArgs...)
		if err != nil {
			return nil, fmt.Errorf("execute local OpenBao command: %w", &redactedOpenBaoCommandError{err: err, token: token})
		}
		return out, nil
	}

	return runCommand(ctx, "kubectl", map[string]string{"KUBECONFIG": kubeconfigPath}, input, args...)
}

type redactedOpenBaoCommandError struct {
	err   error
	token string
}

func (e *redactedOpenBaoCommandError) Error() string {
	return redactOpenBaoToken(e.err.Error(), e.token)
}

func (e *redactedOpenBaoCommandError) Unwrap() error {
	return e.err
}

func redactOpenBaoToken(message, token string) string {
	if token == "" {
		return message
	}
	return strings.ReplaceAll(message, token, "<redacted>")
}

func localOpenBaoExecArgs(token string, interactive bool, targetCmd ...string) []string {
	execArgs := []string{
		"-n", localOpenBaoNamespace,
		"exec",
	}
	if interactive {
		execArgs = append(execArgs, "-i")
	}
	execArgs = append(execArgs,
		localOpenBaoPodName,
		"-c", "openbao",
		"--",
		"env", "BAO_TOKEN="+token,
	)
	return append(execArgs, targetCmd...)
}

func getOpenBaoRootToken(ctx context.Context, kubeconfigPath string) (string, error) {
	out, err := runCommand(ctx, "kubectl", map[string]string{"KUBECONFIG": kubeconfigPath}, "",
		"-n", localOpenBaoNamespace,
		"exec", localOpenBaoPodName,
		"-c", "openbao",
		"--",
		"sh", "-c", `tr -d '\n\r' < /openbao/data/local-bootstrap/init.json | sed -n 's/.*"root_token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'`,
	)
	if err != nil {
		return "", fmt.Errorf("failed to extract OpenBao root token: %w", err)
	}

	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("extracted OpenBao root token is empty")
	}

	return token, nil
}

func writeOpenBaoKVSecret(ctx context.Context, kubeconfigPath, remoteKey string, fields map[string]string) error {
	args := []string{
		"-n", localOpenBaoNamespace,
		"exec", localOpenBaoPodName,
		"--",
		"bao", "kv", "put", fmt.Sprintf("kv/%s", remoteKey),
	}
	for key, value := range fields {
		args = append(args, fmt.Sprintf("%s=%s", key, value))
	}
	if _, err := kubectlExec(ctx, kubeconfigPath, args...); err != nil {
		return fmt.Errorf("write OpenBao secret %q: %w", remoteKey, err)
	}
	return nil
}

func writeLocalOpenBaoValues(state *LocalState) error {
	content := fmt.Sprintf(`# WARNING: This configuration is for LOCAL DEVELOPMENT ONLY.
# It intentionally stores the unseal keys and initial root token on the persistent volume
# to bypass the manual unseal process across reboots. Do NOT use this in production!
# The encrypted storage backend and unseal credentials reside on the same PVC.

server:
  dev:
    enabled: false
  standalone:
    enabled: true
    config: |
      ui = true
      listener "tcp" {
        address = "[::]:8200"
        tls_disable = 1
      }
      storage "file" {
        path = "/openbao/data/backend"
      }
  dataStorage:
    enabled: true
    size: 2Gi
  ha:
    apiAddr: "%s"
  ingress:
    enabled: true
    ingressClassName: traefik
    activeService: false
    hosts:
      - host: %s
        paths:
          - /
    tls: []
  extraContainers:
    - name: auto-unsealer
      image: '{{ .Values.server.image.registry | default "docker.io" }}/{{ .Values.server.image.repository }}:{{ .Values.server.image.tag | default (trimPrefix "v" .Chart.AppVersion) }}'
      command: ["/bin/sh", "-c"]
      args:
        - |
          set -u
          umask 077

          echo "Starting local auto-unsealer sidecar..."
          mkdir -p /openbao/data/local-bootstrap /openbao/data/backend

          INIT_FILE="/openbao/data/local-bootstrap/init.json"
          TMP_FILE="/openbao/data/local-bootstrap/init.json.tmp"

          compact_file() {
            tr -d '\n\r' < "$1"
          }

          extract_unseal_key() {
            compact_file "$1" | sed -n 's/.*"unseal_keys_b64"[[:space:]]*:[[:space:]]*\[[[:space:]]*"\([^"]*\)".*/\1/p'
          }

          extract_root_token() {
            compact_file "$1" | sed -n 's/.*"root_token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p'
          }

          while true; do
            # 1. Fetch API status safely and record exit code
            STATUS=$(bao status -format=json 2>/dev/null)
            EXIT_CODE=$?

            # Handle API readiness/connectivity errors (Exit codes other than 0 or 2)
            if [ $EXIT_CODE -ne 0 ] && [ $EXIT_CODE -ne 2 ]; then
              echo "Waiting for OpenBao API..."
              sleep 2
              continue
            fi

            # 2. Flatten JSON to single line and parse flags safely with sed
            STATUS_COMPACT="$(printf '%%s' "$STATUS" | tr -d '\n\r')"

            INITIALIZED="$(printf '%%s' "$STATUS_COMPACT" | sed -n 's/.*"initialized"[[:space:]]*:[[:space:]]*\(true\|false\).*/\1/p')"
            SEALED="$(printf '%%s' "$STATUS_COMPACT" | sed -n 's/.*"sealed"[[:space:]]*:[[:space:]]*\(true\|false\).*/\1/p')"

            if [ "$INITIALIZED" != "true" ] && [ "$INITIALIZED" != "false" ]; then
              echo "FATAL: Could not parse initialized status from OpenBao API." >&2
              exit 1
            fi

            if [ "$SEALED" != "true" ] && [ "$SEALED" != "false" ]; then
              echo "FATAL: Could not parse sealed status from OpenBao API." >&2
              exit 1
            fi

            # 3. Detect mismatched backend storage / init state
            if [ "$INITIALIZED" = "false" ] && [ -f "$INIT_FILE" ]; then
              echo "FATAL: OpenBao reports initialized=false, but bootstrap file ($INIT_FILE) already exists." >&2
              echo "The OpenBao storage backend and local bootstrap state do not match." >&2
              exit 1
            fi

            if [ "$INITIALIZED" = "true" ]; then
              if [ ! -s "$INIT_FILE" ]; then
                echo "FATAL: OpenBao is initialized but bootstrap file ($INIT_FILE) is missing or empty." >&2
                exit 1
              fi

              CHECK_KEY="$(extract_unseal_key "$INIT_FILE")"
              if [ -z "$CHECK_KEY" ]; then
                echo "FATAL: OpenBao is initialized but bootstrap file does not contain a valid unseal key." >&2
                unset CHECK_KEY
                exit 1
              fi
              unset CHECK_KEY
            fi

            # 4. Handle Initialization (Only when initialized=false and init.json does not exist)
            if [ "$INITIALIZED" = "false" ]; then
              echo "Initializing OpenBao..."
              rm -f "$TMP_FILE"

              if ! bao operator init -key-shares=1 -key-threshold=1 -format=json > "$TMP_FILE"; then
                echo "FATAL: bao operator init failed." >&2
                rm -f "$TMP_FILE"
                exit 1
              fi

              if [ ! -s "$TMP_FILE" ]; then
                echo "FATAL: Initialization output file is empty." >&2
                rm -f "$TMP_FILE"
                exit 1
              fi

              TMP_KEY="$(extract_unseal_key "$TMP_FILE")"
              TMP_TOKEN="$(extract_root_token "$TMP_FILE")"

              if [ -z "$TMP_KEY" ] || [ -z "$TMP_TOKEN" ]; then
                echo "FATAL: Generated initialization data is invalid or missing required keys." >&2
                unset TMP_KEY TMP_TOKEN
                rm -f "$TMP_FILE"
                exit 1
              fi

              unset TMP_KEY TMP_TOKEN
              chmod 600 "$TMP_FILE"
              mv "$TMP_FILE" "$INIT_FILE"

              echo "OpenBao initialization completed."
              sleep 1
              continue
            fi

            # 5. Handle Unsealing (When initialized=true and sealed=true)
            if [ "$SEALED" = "true" ]; then
              UNSEAL_KEY="$(extract_unseal_key "$INIT_FILE")"

              if [ -z "$UNSEAL_KEY" ]; then
                echo "FATAL: Failed to extract unseal key from $INIT_FILE." >&2
                exit 1
              fi

              echo "OpenBao is sealed; unsealing..."

              if bao operator unseal "$UNSEAL_KEY" >/dev/null; then
                echo "OpenBao successfully unsealed."
              else
                echo "FATAL: Failed to unseal OpenBao." >&2
                unset UNSEAL_KEY
                exit 1
              fi

              unset UNSEAL_KEY
            fi

            # Keep monitoring loop active for restarts
            sleep 5
          done
      env:
        - name: VAULT_ADDR
          value: "http://127.0.0.1:8200"
        - name: BAO_ADDR
          value: "http://127.0.0.1:8200"
      volumeMounts:
        - name: data
          mountPath: /openbao/data
injector:
  enabled: false
ui:
  enabled: true
`, localOpenBaoExternalAddress(state), state.OpenBaoHost)

	return writeLocalFile(state.OpenBaoValuesPath, content)
}

func writeLocalClusterSecretStore(opts *Options, state *LocalState) error {
	content := fmt.Sprintf(`apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: %s-%s
spec:
  provider:
    vault:
      server: %s
      path: kv
      version: v2
      auth:
        kubernetes:
          mountPath: kubernetes
          role: %s
          serviceAccountRef:
            name: external-secrets
            namespace: external-secrets
`, opts.ClusterConfig.Name, opts.ClusterConfig.Stage, localOpenBaoInternalAddress, localExternalSecretsRole)
	return writeLocalFile(state.ClusterSecretStorePath, content)
}

func writeLocalArgocdValues(opts *Options, state *LocalState) error {
	content := fmt.Sprintf(`bootstrapValues:
  projects:
    %s-%s:
      sourceRepos:
%s
dexRestarter:
  enabled: false
argo-cd:
  dex:
    enabled: false
  redis:
    resources:
      requests:
        memory: 128Mi
      limits:
        memory: 128Mi
  applicationSet:
    resources:
      requests:
        memory: 256Mi
      limits:
        memory: 256Mi
  repoServer:
    resources:
      requests:
        memory: 512Mi
      limits:
        memory: 512Mi
    volumeMounts:
      - name: host-ca-certificates
        mountPath: %s
        readOnly: true
    volumes:
      - name: host-ca-certificates
        hostPath:
          path: %s
          type: File
  controller:
    resources:
      requests:
        memory: 1000Mi
      limits:
        memory: 1000Mi
  server:
    resources:
      requests:
        memory: 256Mi
      limits:
        memory: 256Mi
    ingressGrpc:
      enabled: false
      annotations:
        cert-manager.io/cluster-issuer: ""
      tls: false
    ingress:
      enabled: true
      ingressClassName: traefik
      annotations:
        cert-manager.io/cluster-issuer: ""
      tls: false
  configs:
    cm:
      url: http://%s/argocd
    rbac:
      policy.default: role:admin
      policy.csv: ""
`, opts.ClusterConfig.Name, opts.ClusterConfig.Stage, formatYAMLList(localProjectSourceRepos(opts)),
		localKindHostCACertPath, localKindHostCACertPath, state.BaseHost)
	return writeLocalFile(state.ArgocdValuesPath, content)
}

func writeLocalCertManagerValues(state *LocalState) error {
	content := `letsencrypt: false
clusterIssuer:
  customDefinition:
    apiVersion: cert-manager.io/v1
    kind: ClusterIssuer
    metadata:
      name: selfsigned-root-issuer
    spec:
      selfSigned: {}
`
	return writeLocalFile(state.CertManagerValuesPath, content)
}

func localOpenBaoExternalAddress(state *LocalState) string {
	return fmt.Sprintf("http://%s", state.OpenBaoHost)
}

func writeLocalIngressOverrides(state *LocalState) error {
	if err := writeLocalFile(state.HomerValuesPath, `ingress:
  enabled: true
  annotations: {}
`); err != nil {
		return err
	}

	if err := writeLocalFile(state.PrometheusValuesPath, `kube-prometheus-stack:
  prometheusOperator:
    resources:
      requests:
        memory: 100Mi
      limits:
        memory: 100Mi
    prometheusConfigReloader:
      resources:
        requests:
          memory: 100Mi
        limits:
          memory: 100Mi
  prometheus-node-exporter:
    resources:
      requests:
        memory: 100Mi
      limits:
        memory: 100Mi
  prometheus:
    ingress:
      enabled: true
      annotations: {}
      tls: []
    prometheusSpec:
      resources:
        requests:
          memory: 384Mi
        limits:
          memory: 384Mi
  alertmanager:
    ingress:
      enabled: true
      annotations: {}
      tls: []
    alertmanagerSpec:
      resources:
        requests:
          memory: 128Mi
        limits:
          memory: 128Mi
  grafana:
    ingress:
      enabled: true
      annotations: {}
      tls: []
    resources:
      requests:
        memory: 256Mi
      limits:
        memory: 256Mi
    initChownData:
      resources:
        requests:
          memory: 128Mi
        limits:
          memory: 128Mi
    sidecar:
      resources:
        requests:
          memory: 100Mi
        limits:
          memory: 100Mi
  kube-state-metrics:
    resources:
      requests:
        memory: 100Mi
      limits:
        memory: 100Mi
prometheus-blackbox-exporter:
  resources:
    requests:
      memory: 100Mi
    limits:
      memory: 100Mi
`); err != nil {
		return err
	}

	if err := writeLocalFile(state.KyvernoValuesPath, `policy-reporter:
  ui:
    enabled: true
    ingress:
      enabled: true
      tls: []
      annotations:
        traefik.ingress.kubernetes.io/app-root: /kyverno/#/
        traefik.ingress.kubernetes.io/router.middlewares: kyverno-policy-reporter-strip-kyverno-prefix@kubernetescrd
middleware:
  name: strip-kyverno-prefix
  namespace: kyverno-policy-reporter
  stripPrefix:
    prefixes:
      - /kyverno
`); err != nil {
		return err
	}

	if err := writeLocalFile(state.LonghornValuesPath, `longhorn:
  ingress:
    enabled: true
    annotations: {}
    tls: false
`); err != nil {
		return err
	}

	content := fmt.Sprintf(`traefik:
  api:
    basePath: /traefik
  ingressRoute:
    dashboard:
      enabled: true
      entryPoints:
        - websecure
      matchRule: Host(`+"`%s`"+`) && PathPrefix(`+"`/traefik`"+`)
      tls: {}
`, state.BaseHost)
	return writeLocalFile(state.TraefikOverlayPath, content)
}

func localProjectSourceRepos(opts *Options) []string {
	repos := []string{
		opts.ClusterConfig.ArgoCD.Repo.HTTPS.Components.URL,
		opts.ClusterConfig.ArgoCD.Repo.HTTPS.Configs.URL,
		"https://charts.external-secrets.io/",
		"https://charts.jetstack.io",
		"https://prometheus-community.github.io/helm-charts",
		"ghcr.io/traefik/helm",
		"oci://ghcr.io/traefik/helm",
	}
	if opts.ClusterConfig.ArgoCD.HelmRepo != nil && strings.TrimSpace(opts.ClusterConfig.ArgoCD.HelmRepo.URL) != "" {
		repos = append(repos, opts.ClusterConfig.ArgoCD.HelmRepo.URL)
	}

	seen := make(map[string]struct{}, len(repos))
	result := make([]string, 0, len(repos))
	for _, repo := range repos {
		trimmed := strings.TrimSpace(repo)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func formatYAMLList(values []string) string {
	var b strings.Builder
	for _, value := range values {
		b.WriteString("        - ")
		b.WriteString(strconv.Quote(value))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func writeLocalFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create directory for %q: %w", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write file %q: %w", path, err)
	}
	return nil
}

func removeLocalFileIfExists(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file %q: %w", path, err)
	}
	return nil
}

func runCommand(ctx context.Context, name string, env map[string]string, input string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for key, value := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
		}
	}
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %s failed: %w\nstderr: %s", name, strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}
