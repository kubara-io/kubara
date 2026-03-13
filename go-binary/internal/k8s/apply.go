package k8s

import (
	"context"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
)

// ApplyOptions for server-side apply operations
type ApplyOptions struct {
	FieldManager   string
	ForceConflicts bool
	DryRun         bool
	Validate       bool
}

// DefaultApplyOptions returns default apply options
func DefaultApplyOptions() ApplyOptions {
	return ApplyOptions{
		FieldManager:   "kubara",
		ForceConflicts: true,
		DryRun:         false,
		Validate:       true,
	}
}

// ApplyManifest applies a multi-document YAML/JSON manifest using server-side apply
func (c *Client) ApplyManifest(ctx context.Context, manifest []byte, opts ApplyOptions) error {
	if opts.FieldManager == "" {
		opts.FieldManager = "kubara"
	}

	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(manifest)), 4096)

	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decoding manifest: %w", err)
		}

		if len(obj.Object) == 0 {
			continue // Skip empty documents
		}

		if err := c.applyObject(ctx, obj, opts); err != nil {
			return fmt.Errorf("applying %s %s/%s: %w",
				obj.GetKind(), obj.GetNamespace(), obj.GetName(), err)
		}
	}

	return nil
}

// applyObject applies a single object using server-side apply
func (c *Client) applyObject(ctx context.Context, obj *unstructured.Unstructured, opts ApplyOptions) error {
	// Get GVR from the object
	gvk := obj.GroupVersionKind()

	// Find the REST mapping for this GVK
	gvr, scope, err := c.getGVR(gvk)
	if err != nil {
		return fmt.Errorf("getting GVR for %s: %w", gvk.String(), err)
	}

	// Get the appropriate resource interface
	var dr dynamic.ResourceInterface
	if scope == meta.RESTScopeNamespace {
		if obj.GetNamespace() == "" {
			obj.SetNamespace("default")
		}
		dr = c.DynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
	} else {
		dr = c.DynamicClient.Resource(gvr)
	}

	// Prepare apply options
	applyOpts := metav1.ApplyOptions{
		FieldManager: opts.FieldManager,
		Force:        opts.ForceConflicts,
	}

	if opts.DryRun {
		applyOpts.DryRun = []string{metav1.DryRunAll}
	}

	// Server-side apply
	_, err = dr.Apply(ctx, obj.GetName(), obj, applyOpts)
	if err != nil {
		return fmt.Errorf("server-side apply failed: %w", err)
	}

	return nil
}

// getGVR gets GroupVersionResource from GroupVersionKind
func (c *Client) getGVR(gvk schema.GroupVersionKind) (schema.GroupVersionResource, meta.RESTScope, error) {
	// Use the REST mapper to find the GVR
	mapping, err := c.RESTMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, nil, fmt.Errorf("REST mapping for %s: %w", gvk.String(), err)
	}

	return mapping.Resource, mapping.Scope, nil
}

// FilterCRDs extracts only CustomResourceDefinition objects from a manifest
func FilterCRDs(manifest []byte) ([]byte, error) {
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(manifest)), 4096)
	var crds []string

	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decoding manifest: %w", err)
		}

		if len(obj.Object) == 0 {
			continue
		}

		if obj.GetKind() == "CustomResourceDefinition" {
			// Convert back to YAML
			yamlData, err := runtime.Encode(unstructured.UnstructuredJSONScheme, obj)
			if err != nil {
				return nil, fmt.Errorf("encoding CRD: %w", err)
			}
			crds = append(crds, string(yamlData))
		}
	}

	return []byte(strings.Join(crds, "---\n")), nil
}
