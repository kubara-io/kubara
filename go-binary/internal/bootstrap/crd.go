package bootstrap

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"kubara/internal/helm"
	"kubara/internal/k8s"

	"github.com/rs/zerolog/log"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// CRDManager handles CustomResourceDefinition operations for bootstrap
type CRDManager struct {
	client    *k8s.Client
	extClient apiextensionsclient.Interface
	crdCache  map[string][]string // Cache: chartPath -> CRD names
}

// NewCRDManager creates a new CRD manager
func NewCRDManager(client *k8s.Client) *CRDManager {
	extClient, err := apiextensionsclient.NewForConfig(client.RESTConfig)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create k8s API extensions client, CRD operations will be limited")
	}
	return &CRDManager{
		client:    client,
		extClient: extClient,
		crdCache:  make(map[string][]string),
	}
}

// ApplyChartCRDs applies CRDs from a helm chart
func (cm *CRDManager) ApplyChartCRDs(ctx context.Context, chartPath string, dryRun bool, apiVersions []string) error {
	// Template the chart with CRDs included
	manifest, err := helm.Template(ctx, helm.TemplateOptions{
		ReleaseName: "crd-template",
		ChartPath:   chartPath,
		IncludeCRDs: true,
		APIVersions: apiVersions,
	})
	if err != nil {
		return fmt.Errorf("templating chart for CRDs: %w", err)
	}

	// Filter for CRDs only
	crdManifest, err := k8s.FilterCRDs(manifest)
	if err != nil {
		return fmt.Errorf("filtering CRDs: %w", err)
	}

	// Cache empty result first to avoid re-templating
	if len(crdManifest) == 0 {
		fmt.Println("No CRDs found in chart")
		// Cache empty result to avoid re-templating
		cm.crdCache[chartPath] = []string{}
		return nil
	}

	// Extract and cache CRD names only if manifest is not empty
	crdNames, err := cm.extractCRDNames(crdManifest)
	if err != nil {
		return fmt.Errorf("extracting CRD names: %w", err)
	}

	// Cache the CRD names for this chart
	cm.crdCache[chartPath] = crdNames

	// Apply CRDs with server-side apply
	opts := k8s.DefaultApplyOptions()
	opts.FieldManager = "kubara-bootstrap-crd"
	opts.ForceConflicts = true
	opts.DryRun = dryRun

	return cm.client.ApplyManifest(ctx, crdManifest, opts)
}

// WaitForCRDs waits for CRDs to be established
func (cm *CRDManager) WaitForCRDs(ctx context.Context, crdNames []string) error {
	if len(crdNames) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(crdNames))
	semaphore := make(chan struct{}, 5)

	for _, crdName := range crdNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			if err := cm.waitForCRD(ctx, name); err != nil {
				errChan <- fmt.Errorf("waiting for CRD %s: %w", name, err)
			}
		}(crdName)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// waitForCRD waits for a single CRD to be established
func (cm *CRDManager) waitForCRD(ctx context.Context, crdName string) error {
	if cm.extClient == nil {
		return fmt.Errorf("k8s API extensions client not available")
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for CRD %s to be established", crdName)
		case <-ticker.C:
			// Check if CRD exists and is established
			crd, err := cm.extClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crdName, v1.GetOptions{})
			if err != nil {
				continue
			}
			// Check if CRD is established
			for _, condition := range crd.Status.Conditions {
				if condition.Type == apiextensionsv1.Established && condition.Status == apiextensionsv1.ConditionTrue {
					return nil
				}
			}
		}
	}
}

// GetChartCRDNames extracts CRD names from a helm chart
func (cm *CRDManager) GetChartCRDNames(ctx context.Context, chartPath string) ([]string, error) {
	// Check cache first
	if crdNames, exists := cm.crdCache[chartPath]; exists {
		return crdNames, nil
	}

	// Fallback to templating if not in cache
	manifest, err := helm.Template(ctx, helm.TemplateOptions{
		ReleaseName: "crd-discovery",
		ChartPath:   chartPath,
		IncludeCRDs: true,
	})
	if err != nil {
		return nil, fmt.Errorf("templating chart for CRD discovery: %w", err)
	}

	crdManifest, err := k8s.FilterCRDs(manifest)
	if err != nil {
		return nil, fmt.Errorf("filtering CRDs: %w", err)
	}

	crdNames, err := cm.extractCRDNames(crdManifest)
	if err != nil {
		return nil, fmt.Errorf("extracting CRD names: %w", err)
	}

	// Cache the result
	cm.crdCache[chartPath] = crdNames
	return crdNames, nil
}

func (cm *CRDManager) extractCRDNames(crdManifest []byte) ([]string, error) {
	// Handle empty manifest
	if len(crdManifest) == 0 {
		return []string{}, nil
	}

	var crdNames []string
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(crdManifest)), 4096)

	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decoding CRD: %w", err)
		}

		if len(obj.Object) > 0 && obj.GetKind() == "CustomResourceDefinition" {
			crdNames = append(crdNames, obj.GetName())
		}
	}

	return crdNames, nil
}
