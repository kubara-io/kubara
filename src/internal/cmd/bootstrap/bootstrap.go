package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/envconfig"
	"github.com/kubara-io/kubara/internal/helm"
	"github.com/kubara-io/kubara/internal/k8s"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

const (
	prometheusAPIVersion     = "monitoring.coreos.com/v1"
	argocdNamespace          = "argocd"
	externalSecretsNamespace = "external-secrets"
)

// Options for bootstrap operations
type Options struct {
	Kubeconfig         string
	PlatformComponents string
	PlatformConfigs    string
	WithES             bool
	WithProm           bool
	Local              bool
	WithESCSSPath      string
	EnvMap             *envconfig.EnvMap
	Catalog            catalog.Catalog
	ClusterConfig      *config.Cluster
	DryRun             bool
	Timeout            time.Duration
	ClusterName        string
	WorkDir            string
	ConfigFilePath     string
	CatalogPath        string
	CatalogOverwrite   bool
	LocalState         *LocalState
}

type BootstrapChart struct {
	Name            string
	Namespace       string
	Path            string
	OverlayValues   []string
	RepoURL         string
	Enabled         bool
	EnsureNamespace bool
	EnsureCRD       bool
}

type LocalState struct {
	RuntimeDir             string
	KubeconfigPath         string
	KindConfigPath         string
	LoadBalancerIP         string
	BaseHost               string
	OpenBaoHost            string
	TraefikOverlayPath     string
	CertManagerValuesPath  string
	OpenBaoValuesPath      string
	ArgocdValuesPath       string
	HomerValuesPath        string
	PrometheusValuesPath   string
	KyvernoValuesPath      string
	LonghornValuesPath     string
	OAuth2ProxyValuesPath  string
	ClusterSecretStorePath string
	GenerateEnvPath        string
}

// Bootstrap orchestrates the complete ArgoCD bootstrap process
func Bootstrap(ctx context.Context, opts *Options) error {
	if opts.Local && opts.DryRun {
		return fmt.Errorf("--dry-run is not supported together with --local")
	}
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	if opts.Local {
		if err := prepareLocalBootstrap(ctx, opts); err != nil {
			return fmt.Errorf("prepare local bootstrap: %w", err)
		}
		if opts.LocalState == nil || opts.LocalState.KubeconfigPath == "" {
			return fmt.Errorf("local bootstrap did not provide a kubeconfig path")
		}
		opts.Kubeconfig = opts.LocalState.KubeconfigPath
	}

	// Create Kubernetes client
	client, err := k8s.NewClient(k8s.Config{
		KubeconfigPath: opts.Kubeconfig,
		QPS:            50,
		Burst:          100,
		Timeout:        30 * time.Second,
		UserAgent:      "kubara-bootstrap",
	})
	if err != nil {
		return fmt.Errorf("create kubernetes client: %w", err)
	}

	argocdChartPath, err := chartPathForService(opts.Catalog, "argocd")
	if err != nil {
		return err
	}
	externalSecretsChartPath, err := chartPathForService(opts.Catalog, "external-secrets")
	if err != nil {
		return err
	}
	prometheusChartPath, err := chartPathForService(opts.Catalog, "kube-prometheus-stack")
	if err != nil {
		return err
	}

	// Construct bootstrapCharts structs
	bootstrapCharts := []BootstrapChart{
		{
			Name:            "argocd",
			Namespace:       argocdNamespace,
			Path:            filepath.Join(opts.PlatformComponents, "helm", argocdChartPath),
			OverlayValues:   overlayValuesForChart(opts, argocdChartPath),
			RepoURL:         "https://argoproj.github.io/argo-helm",
			Enabled:         true,
			EnsureNamespace: true,
			EnsureCRD:       true,
		},
		{
			Name:            "external-secrets",
			Namespace:       externalSecretsNamespace,
			Path:            filepath.Join(opts.PlatformComponents, "helm", externalSecretsChartPath),
			OverlayValues:   overlayValuesForChart(opts, externalSecretsChartPath),
			RepoURL:         "https://charts.external-secrets.io",
			Enabled:         opts.WithES,
			EnsureNamespace: opts.WithES,
			EnsureCRD:       opts.WithES,
		},
		{
			Name:            "kube-prometheus-stack",
			Path:            filepath.Join(opts.PlatformComponents, "helm", prometheusChartPath),
			OverlayValues:   overlayValuesForChart(opts, prometheusChartPath),
			RepoURL:         "https://prometheus-community.github.io/helm-charts",
			Enabled:         opts.WithProm,
			EnsureNamespace: false,
			EnsureCRD:       opts.WithProm,
		},
	}

	// Locate ArgoChart for later use
	var argoChart BootstrapChart
	for _, c := range bootstrapCharts {
		if c.Name == "argocd" {
			argoChart = c
			break
		}
	}

	log.Info().Msg("Starting bootstrap process")

	// Step 1: Ensure namespaces exist
	if err := ensureNamespaces(ctx, client, opts, bootstrapCharts); err != nil {
		return fmt.Errorf("ensure namespaces: %w", err)
	}

	// Step 2: Add helm repositories
	if err := addHelmRepositories(ctx, bootstrapCharts); err != nil {
		return fmt.Errorf("add helm repositories: %w", err)
	}

	// Step 3: Update helm repositories
	if err := updateHelmDependencies(ctx, opts, bootstrapCharts); err != nil {
		return fmt.Errorf("update helm repositories: %w", err)
	}

	// Step 4: Apply CRDs
	if err := applyCRDs(ctx, client, opts, bootstrapCharts); err != nil {
		return fmt.Errorf("apply CRDs: %w", err)
	}
	// Refresh discovery/REST mapper so CRDs installed above are visible for
	// subsequent server-side apply calls in the same bootstrap run.
	client.RefreshDiscovery()

	// Step 5: Apply secrets before ArgoCD
	if err := applySecrets(ctx, client, opts); err != nil {
		return fmt.Errorf("apply secrets: %w", err)
	}

	// Step 6: Bootstrap ArgoCD
	if err := bootstrapArgoCD(ctx, client, opts, argoChart); err != nil {
		return fmt.Errorf("bootstrap ArgoCD: %w", err)
	}

	// Step 7: Wait for ArgoCD to be ready
	if err := waitForArgoCD(ctx, client, opts, argoChart); err != nil {
		return fmt.Errorf("wait for ArgoCD readiness: %w", err)
	}

	// Step 8: Print completion message
	printCompletionMessage(opts)
	log.Info().Msg("ArgoCD bootstrap completed successfully")
	return nil
}

func chartPathForService(cat catalog.Catalog, serviceName string) (string, error) {
	definition, exists := cat.Services[serviceName]
	if !exists {
		return "", fmt.Errorf("bootstrap service %q is missing from catalog", serviceName)
	}

	return definition.Spec.ChartPath, nil
}

func overlayValuesForChart(opts *Options, chartPath string) []string {
	chartOverlayPath := filepath.Join(opts.PlatformConfigs, opts.ClusterName, "helm", chartPath)
	valuesPaths := []string{filepath.Join(chartOverlayPath, "values.generated.yaml")}

	extraValues, err := filepath.Glob(filepath.Join(chartOverlayPath, "values-*.yaml"))
	if err == nil {
		sort.Strings(extraValues)
		valuesPaths = append(valuesPaths, extraValues...)
	}
	return valuesPaths
}

// ensureNamespaces ensures required namespaces exist
func ensureNamespaces(ctx context.Context, client *k8s.Client, opts *Options, charts []BootstrapChart) error {
	log.Info().Msg("Ensuring namespaces exist")

	for _, chart := range charts {
		if chart.EnsureNamespace {
			if err := client.EnsureNamespace(ctx, chart.Namespace, opts.DryRun); err != nil {
				return fmt.Errorf("ensure %q namespace: %w", chart.Name, err)
			}
		}
	}

	return nil
}

// addHelmRepositories adds required helm repositories
func addHelmRepositories(ctx context.Context, charts []BootstrapChart) error {
	log.Info().Msg("Adding helm repositories")

	for _, chart := range charts {
		if chart.Enabled {
			repo := helm.RepoOptions{Name: chart.Name, URL: chart.RepoURL}
			if err := helm.AddRepository(ctx, repo); err != nil {
				return fmt.Errorf("add helm repository %q: %w", repo.Name, err)
			}

			log.Info().Msgf("Added helm repository: %q", repo.Name)
		}
	}

	// Refresh the global repository index too. Per-alias updates can leave the
	// shared cache out of date for charts whose Chart.yaml references the
	// repository by URL rather than by the alias added above, which triggers
	// "can't get a valid version for subchart" errors on the next
	// helm dependency build.
	if err := helm.UpdateAllRepositories(ctx); err != nil {
		return fmt.Errorf("refresh helm repository index: %w", err)
	}
	log.Info().Msg("Refreshed helm repository index")

	return nil
}

// updateHelmDependencies updates dependencies for required charts
func updateHelmDependencies(ctx context.Context, opts *Options, charts []BootstrapChart) error {
	log.Info().Msg("Updating helm chart dependencies")

	for _, chart := range charts {
		if chart.Enabled {
			dep := helm.DependencyOptions{ChartPath: chart.Path, Timeout: opts.Timeout}
			if err := helm.BuildDependencies(ctx, dep); err != nil {
				return fmt.Errorf("build helm chart dependencies for %q: %w", chart.Name, err)
			}
			log.Info().Msgf("Updated helm dependencies for chart: %q", chart.Name)

		}
	}
	return nil
}

// applyCRDs applies CustomResourceDefinitions from charts
func applyCRDs(ctx context.Context, client *k8s.Client, opts *Options, charts []BootstrapChart) error {
	log.Info().Msg("Applying CRDs")

	crdManager := NewCRDManager(client)

	for _, chart := range charts {
		if !chart.EnsureCRD {
			continue
		}

		if err := crdManager.ApplyChartCRDs(ctx, chart.Path, opts.DryRun, []string{
			prometheusAPIVersion, // For prometheus-operator CRDs
		}); err != nil {
			return fmt.Errorf("applying CRDs for %q: %w", chart.Name, err)
		}

		// Get CRD names and wait for them to be established
		crdNames, err := crdManager.GetChartCRDNames(ctx, chart.Path)
		if err != nil {
			log.Warn().Err(err).Msgf("Could not get CRD names for %q, skipping wait", chart.Name)
			continue
		}

		// Skip waiting if we are in dry-run mode
		if opts.DryRun {
			log.Info().Msgf("[DRY-RUN] Skipping wait for CRDs: %q", chart.Name)
			continue
		}

		if len(crdNames) > 0 {
			if err := crdManager.WaitForCRDs(ctx, crdNames); err != nil {
				return fmt.Errorf("waiting for CRDs from %q: %w", chart.Name, err)
			}
			log.Info().Msgf("CRDs applied and established for: %q", chart.Name)
		}
	}

	return nil
}

// bootstrapArgoCD performs the main ArgoCD installation
func bootstrapArgoCD(ctx context.Context, client *k8s.Client, opts *Options, argoChart BootstrapChart) error {
	log.Info().Msg("Bootstrapping ArgoCD")

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(opts.EnvMap.ArgocdWizardAccountPassword), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Template ArgoCD with hashed password
	manifest, err := helm.Template(ctx, helm.TemplateOptions{
		ReleaseName: argoChart.Namespace,
		ChartPath:   argoChart.Path,
		Namespace:   argoChart.Namespace,
		ValuesPaths: argoChart.OverlayValues,
		APIVersions: []string{prometheusAPIVersion},
		SetArgs: []string{
			fmt.Sprintf("argo-cd.configs.secret.extra.accounts\\.wizard\\.password=%s", string(hashedPassword)),
		},
	})
	if err != nil {
		return fmt.Errorf("template ArgoCD: %w", err)
	}

	// Apply ArgoCD manifest
	applyOpts := k8s.DefaultApplyOptions()
	applyOpts.FieldManager = "kubara-argocd-bootstrap"
	applyOpts.ForceConflicts = true

	// TODO: Implement proper DryRun with client
	if opts.DryRun {
		log.Info().Msg("DRY RUN: Would apply ArgoCD manifest")
		return nil
	}

	if err := client.ApplyManifest(ctx, manifest, applyOpts); err != nil {
		return fmt.Errorf("apply ArgoCD manifest: %w", err)
	}

	log.Info().Msg("ArgoCD manifest applied successfully")
	return nil
}

// waitForArgoCD waits for ArgoCD components to be ready
func waitForArgoCD(ctx context.Context, client *k8s.Client, opts *Options, argoChart BootstrapChart) error {
	if opts.DryRun {
		log.Info().Msg("Skipping wait because dry-run is set")
		return nil
	}
	log.Info().Msg("Waiting for ArgoCD components to be ready")

	// Wait for ArgoCD server pod
	if err := client.WaitForPod(ctx, argoChart.Namespace, "app.kubernetes.io/name=argocd-server"); err != nil {
		return fmt.Errorf("wait for ArgoCD server: %w", err)
	}

	// Wait for ArgoCD repo server pod
	if err := client.WaitForPod(ctx, argoChart.Namespace, "app.kubernetes.io/name=argocd-repo-server"); err != nil {
		return fmt.Errorf("wait for ArgoCD repo server: %w", err)
	}

	// Wait for ArgoCD deployment to be ready
	if err := client.WaitForDeployment(ctx, argoChart.Namespace, "argocd-server"); err != nil {
		return fmt.Errorf("wait for ArgoCD deployment: %w", err)
	}

	log.Info().Msg("ArgoCD components are ready")
	return nil
}

// applySecrets applies the bootstrap secrets
func applySecrets(ctx context.Context, client *k8s.Client, opts *Options) error {
	log.Info().Msg("Applying secrets")

	secretManager := NewSecretManager(client)

	// Apply hub cluster secrets
	if err := secretManager.CreateHubSecrets(ctx, opts); err != nil {
		return fmt.Errorf("apply hub cluster secrets: %w", err)
	}

	log.Info().Msg("Secrets applied successfully")
	return nil
}

// CompletionLogConfig contains the data needed to render the bootstrap completion output.
type CompletionLogConfig struct {
	WizardPassword string
	ClusterDNSName string
	Local          bool
	OpenBaoHost    string
}

// printCompletionMessage prints the completion message with access instructions
func printCompletionMessage(opts *Options) {
	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] ArgoCD bootstrap completed successfully")
	} else {
		config := CompletionLogConfig{}
		if opts.ClusterConfig != nil {
			config.ClusterDNSName = opts.ClusterConfig.DNSName
		}
		config.WizardPassword = opts.EnvMap.ArgocdWizardAccountPassword
		config.Local = opts.Local
		if opts.LocalState != nil {
			config.OpenBaoHost = opts.LocalState.OpenBaoHost
		}
		log.Info().Msg(CreateCompletionMessage(config))
	}
}

func completionIngressHost(config CompletionLogConfig) string {
	return config.ClusterDNSName
}

// CreateCompletionMessage returns the formatted completion message.
func CreateCompletionMessage(config CompletionLogConfig) string {
	if config.Local {
		return fmt.Sprintf(`
🎉 Local kubara bootstrap complete!

📝 Next steps:
1. Commit and push all repository contents generated by kubara.
2. Ensure the Git repository is reachable by Argo CD.
   If you did not provide both ARGOCD_GIT_USERNAME and ARGOCD_GIT_PAT_OR_PASSWORD, ensure the Git repository is public so Argo CD can pull it.

Argo CD should be reachable in a couple of minutes via:
    https://%s/argocd

You can otherwise immediately access the Argo CD UI via:
    kubectl --kubeconfig .local/kind.kubeconfig port-forward svc/argocd-server -n argocd 8080:443
    then open http://localhost:8080/argocd in your browser

Log in with:
    wizard / %s

Other useful links:
  - Portal:  https://%s
  - OpenBao: https://%s/ui login with root

`, config.ClusterDNSName, config.WizardPassword, config.ClusterDNSName, config.OpenBaoHost)
	}

	formattedOutput := ""
	ingressHost := completionIngressHost(config)
	if ingressHost != "" {
		formattedOutput = fmt.Sprintf(" or try: https://%s/argocd (if ingress is running)", ingressHost)
	}

	return fmt.Sprintf(`
🎉 ArgoCD bootstrap complete!

You can access the Argo CD UI with user "wizard" and your chosen password "%s" at:

    kubectl port-forward svc/argocd-server -n argocd 8080:443 --kubeconfig ...

Then open: http://localhost:8080/argocd%s

📝 Next steps:
1. Log in with username: wizard
2. Configure your applications
3. Set up monitoring and logging as needed`, config.WizardPassword, formattedOutput)
}
