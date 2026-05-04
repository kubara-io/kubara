package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

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
	Kubeconfig     string
	ManagedCatalog string
	OverlayValues  string
	WithES         bool
	WithProm       bool
	WithESCSSPath  string
	EnvMap         *envconfig.EnvMap
	ClusterConfig  *config.Cluster
	DryRun         bool
	Timeout        time.Duration
	ClusterName    string
}

type BootstrapChart struct {
	Name            string
	Namespace       string
	Path            string
	OverlayValues   []string
	RepoURL         string
	EnsureNamespace bool
	EnsureCRD       bool
}

// Bootstrap orchestrates the complete ArgoCD bootstrap process
func Bootstrap(ctx context.Context, opts *Options) error {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
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

	// Construct bootstrapCharts structs
	bootstrapCharts := []BootstrapChart{
		{
			Name:            "argo-cd",
			Namespace:       argocdNamespace,
			Path:            filepath.Join(opts.ManagedCatalog, "helm", "argo-cd"),
			OverlayValues:   []string{filepath.Join(opts.OverlayValues, "helm", opts.ClusterName, "argo-cd", "values.yaml")},
			RepoURL:         "https://argoproj.github.io/argo-helm",
			EnsureNamespace: true,
			EnsureCRD:       true,
		},
		{
			Name:            "external-secrets",
			Namespace:       externalSecretsNamespace,
			Path:            filepath.Join(opts.ManagedCatalog, "helm", "external-secrets"),
			OverlayValues:   []string{filepath.Join(opts.OverlayValues, "helm", opts.ClusterName, "external-secrets", "values.yaml")},
			RepoURL:         "https://charts.external-secrets.io",
			EnsureNamespace: opts.WithES,
			EnsureCRD:       opts.WithES,
		},
		{
			Name:            "kube-prometheus-stack",
			Path:            filepath.Join(opts.ManagedCatalog, "helm", "kube-prometheus-stack"),
			OverlayValues:   []string{filepath.Join(opts.OverlayValues, "helm", opts.ClusterName, "kube-prometheus-stack", "values.yaml")},
			RepoURL:         "https://prometheus-community.github.io/helm-charts",
			EnsureNamespace: false,
			EnsureCRD:       opts.WithProm,
		},
	}

	// Locate ArgoChart for later use
	var argoChart BootstrapChart
	for _, c := range bootstrapCharts {
		if c.Name == "argo-cd" {
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
		if chart.EnsureCRD {
			repo := helm.RepoOptions{Name: chart.Name, URL: chart.RepoURL}
			if err := helm.AddRepository(ctx, repo); err != nil {
				return fmt.Errorf("add helm repository %q: %w", repo.Name, err)
			}

			if err := helm.UpdateRepository(ctx, repo); err != nil {
				return fmt.Errorf("update helm repository %q: %w", repo.Name, err)
			}
			log.Info().Msgf("Added helm repository: %q", repo.Name)
		}
	}

	return nil
}

// updateHelmDependencies updates dependencies for required charts
func updateHelmDependencies(ctx context.Context, opts *Options, charts []BootstrapChart) error {
	log.Info().Msg("Updating helm chart dependencies")

	for _, chart := range charts {
		if chart.EnsureCRD {
			dep := helm.DependencyOptions{ChartPath: chart.Path, Timeout: opts.Timeout}
			if err := helm.UpdateDependencies(ctx, dep); err != nil {
				return fmt.Errorf("update helm chart dependencies for %q: %w", chart.Name, err)
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

	// Apply control plane secrets
	if err := secretManager.CreateControlPlaneSecrets(ctx, opts); err != nil {
		return fmt.Errorf("apply control plane secrets: %w", err)
	}

	log.Info().Msg("Secrets applied successfully")
	return nil
}

// CompletionLogConfig contains the data needed to render the bootstrap completion output.
type CompletionLogConfig struct {
	WizardPassword string
	ClusterDNSName string
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
		log.Info().Msg(CreateCompletionMessage(config))
	}
}

func completionIngressHost(config CompletionLogConfig) string {
	return config.ClusterDNSName
}

// CreateCompletionMessage returns the formatted completion message.
func CreateCompletionMessage(config CompletionLogConfig) string {
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
