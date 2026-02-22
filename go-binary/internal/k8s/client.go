package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
)

// Client wraps Kubernetes clients with enhanced functionality
type Client struct {
	Clientset     kubernetes.Interface
	DynamicClient dynamic.Interface
	Discovery     discovery.CachedDiscoveryInterface
	RESTConfig    *rest.Config
	RESTMapper    meta.RESTMapper
}

// Config options for client creation
type Config struct {
	KubeconfigPath string
	QPS            int32
	Burst          int32
	Timeout        time.Duration
	UserAgent      string
}

// NewClient creates a new Kubernetes client
func NewClient(cfg Config) (*Client, error) {
	// Build config from kubeconfig
	restConfig, err := buildRESTConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("building REST config: %w", err)
	}

	// Apply client configuration
	applyClientConfig(restConfig, cfg)

	// Create clients with proper scheme initialization
	if err := initScheme(); err != nil {
		return nil, fmt.Errorf("initializing scheme: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	// Create discovery client with caching
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating discovery client: %w", err)
	}
	cachedDiscovery := memory.NewMemCacheClient(discoveryClient)

	// Create REST mapper
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscovery)

	return &Client{
		Clientset:     clientset,
		DynamicClient: dynamicClient,
		Discovery:     cachedDiscovery,
		RESTConfig:    restConfig,
		RESTMapper:    restMapper,
	}, nil
}

// RefreshDiscovery invalidates discovery cache and resets the REST mapper so
// newly installed CRDs are discoverable within the same process.
func (c *Client) RefreshDiscovery() {
	if c == nil {
		return
	}

	if c.Discovery != nil {
		c.Discovery.Invalidate()
	}

	type resettableMapper interface {
		Reset()
	}

	if mapper, ok := c.RESTMapper.(resettableMapper); ok {
		mapper.Reset()
	}
}

// initScheme initializes the scheme with CRD support
func initScheme() error {
	_ = apiextensions.AddToScheme(scheme.Scheme)
	return nil
}

// buildRESTConfig builds REST config with enhanced kubeconfig resolution
func buildRESTConfig(cfg Config) (*rest.Config, error) {
	var kubeconfigPath string

	// TODO: use correct vars from bootstrap command opts
	if cfg.KubeconfigPath != "" && cfg.KubeconfigPath != "~/.kube/config" {
		kubeconfigPath = cfg.KubeconfigPath
	} else {
		// Try KUBECONFIG env var first
		if envKC := os.Getenv("KUBECONFIG"); envKC != "" {
			kubeconfigPath = envKC
		} else {
			// Default to ~/.kube/config
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("getting home directory: %w", err)
			}
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfigPath
	configOverrides := &clientcmd.ConfigOverrides{}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	).ClientConfig()
}

// applyClientConfig applies rate limiting and timeout settings
func applyClientConfig(config *rest.Config, cfg Config) {
	// Set user agent
	if cfg.UserAgent != "" {
		config.UserAgent = cfg.UserAgent
	} else {
		config.UserAgent = "kubara/1.0.0"
	}

	// Set rate limiting
	if cfg.QPS > 0 {
		config.QPS = float32(cfg.QPS)
	} else {
		config.QPS = 100.0
	}

	if cfg.Burst > 0 {
		config.Burst = int(cfg.Burst)
	} else {
		config.Burst = 200
	}

	// Set timeout
	if cfg.Timeout > 0 {
		config.Timeout = cfg.Timeout
	} else {
		config.Timeout = 60 * time.Second
	}

	// Configure rate limiter with enhanced flow control
	config.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(config.QPS, config.Burst)
}
