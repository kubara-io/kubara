package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"

	"github.com/kubara-io/kubara/internal/catalog"
	"github.com/kubara-io/kubara/internal/config"
	"github.com/kubara-io/kubara/internal/k8s"
	"github.com/kubara-io/kubara/internal/spoke"
	"github.com/rs/zerolog/log"
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

	if err := prepareSpokeCluster(ctx, opts); err != nil {
		return err
	}

	if err := spoke.UpsertBootstrapSecret(ctx, client, opts.ClusterName, initialKubeconfig); err != nil {
		return err
	}

	log.Info().Msgf("Waiting for credential rotator to onboard spoke %q...", opts.ClusterName)

	if err := spoke.WaitForOnboarding(ctx, client, opts.ClusterName); err != nil {
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
