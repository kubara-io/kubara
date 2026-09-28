package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		return nil, fmt.Errorf("build REST config: %w", err)
	}

	// Apply client configuration
	applyClientConfig(restConfig, cfg)

	// Create clients with proper scheme initialization
	if err := initScheme(); err != nil {
		return nil, fmt.Errorf("initialize scheme: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}

	// Create discovery client with caching
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create discovery client: %w", err)
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
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	path := strings.TrimSpace(cfg.KubeconfigPath)
	if path != "" {
		if path == "~" || strings.HasPrefix(path, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("get home directory: %w", err)
			}
			if path == "~" {
				path = home
			} else {
				path = filepath.Join(home, path[2:])
			}
		}
		loadingRules.ExplicitPath = path
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
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
