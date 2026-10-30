package cluster

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kubara-io/kubara/internal/k8s"
	"github.com/kubara-io/kubara/internal/spoke"
	"github.com/kubara-io/kubara/internal/utils"
	"github.com/urfave/cli/v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func CreateRotateCredentialsCommand() *cli.Command {
	var (
		tokenDuration = time.Hour
		refreshBefore = 15 * time.Minute
		clusterRole   = spoke.DefaultAgentClusterRoleName
		argoEnabled   = false
		argoNamespace = "argocd"
		fluxEnabled   = false
		fluxNamespace = "flux-system"
	)

	return &cli.Command{
		Name:        "rotate-credentials",
		Usage:       "Reconcile and rotate spoke cluster credentials on the hub",
		UsageText:   "kubara cluster rotate-credentials [flags]",
		Description: "Reconciles spoke cluster credential secrets in kubara-system, mints short-lived tokens, and projects credentials to enabled GitOps targets.",
		Hidden:      true,
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:        "token-duration",
				Value:       tokenDuration,
				Usage:       "Lifetime of minted spoke ServiceAccount tokens",
				Destination: &tokenDuration,
			},
			&cli.DurationFlag{
				Name:        "refresh-before",
				Value:       refreshBefore,
				Usage:       "Rotate spoke tokens when remaining lifetime is at or below this duration",
				Destination: &refreshBefore,
			},
			&cli.StringFlag{
				Name:        "cluster-role",
				Value:       clusterRole,
				Usage:       "ClusterRole bound to kubara-agent on spoke clusters",
				Destination: &clusterRole,
			},
			&cli.BoolFlag{
				Name:        "argo-enabled",
				Value:       argoEnabled,
				Usage:       "Project rotated spoke credentials into the Argo CD namespace",
				Destination: &argoEnabled,
			},
			&cli.StringFlag{
				Name:        "argo-namespace",
				Value:       argoNamespace,
				Usage:       "Target namespace for Argo CD cluster secrets",
				Destination: &argoNamespace,
			},
			&cli.BoolFlag{
				Name:        "flux-enabled",
				Value:       fluxEnabled,
				Usage:       "Project rotated spoke credentials into the Flux namespace",
				Destination: &fluxEnabled,
			},
			&cli.StringFlag{
				Name:        "flux-namespace",
				Value:       fluxNamespace,
				Usage:       "Target namespace for Flux kubeconfig secrets",
				Destination: &fluxNamespace,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			hubClient, err := resolveHubClient(cmd)
			if err != nil {
				return err
			}

			reconciler, err := spoke.NewReconciler(hubClient, spoke.Config{
				TokenDuration: tokenDuration,
				RefreshBefore: refreshBefore,
				ClusterRole:   clusterRole,
				ArgoEnabled:   argoEnabled,
				ArgoNamespace: argoNamespace,
				FluxEnabled:   fluxEnabled,
				FluxNamespace: fluxNamespace,
			})
			if err != nil {
				return err
			}

			return reconciler.ReconcileAll(ctx)
		},
	}
}

func resolveHubClient(cmd *cli.Command) (kubernetes.Interface, error) {
	if !cmd.IsSet("kubeconfig") && os.Getenv("KUBECONFIG") == "" {
		if restCfg, err := rest.InClusterConfig(); err == nil {
			restCfg.Timeout = 30 * time.Second
			clientset, err := kubernetes.NewForConfig(restCfg)
			if err != nil {
				return nil, fmt.Errorf("create in-cluster kubernetes client: %w", err)
			}
			return clientset, nil
		}
	}

	cwd, err := filepath.Abs(cmd.String("work-dir"))
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	kubeconfigPath, err := utils.GetFullPath(cmd.String("kubeconfig"), cwd)
	if err != nil {
		return nil, fmt.Errorf("get kubeconfig path: %w", err)
	}

	client, err := k8s.NewClient(k8s.Config{
		KubeconfigPath: kubeconfigPath,
		QPS:            50,
		Burst:          100,
		Timeout:        30 * time.Second,
		UserAgent:      "kubara-credential-rotator",
	})
	if err != nil {
		return nil, fmt.Errorf("create hub kubernetes client: %w", err)
	}

	return client.Clientset, nil
}
