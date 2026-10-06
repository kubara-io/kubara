package helm

import (
	"fmt"
	"io"
	"sort"
	"strings"

	helmRelease "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// ManifestPlan represents partitioned and ordered Kubernetes manifests according to Helm hook specifications.
type ManifestPlan struct {
	PreInstall  []*unstructured.Unstructured
	Main        []*unstructured.Unstructured
	PostInstall []*unstructured.Unstructured
	PreDelete   []*unstructured.Unstructured
	PostDelete  []*unstructured.Unstructured
}

// PlanManifest decodes a multi-document YAML/JSON manifest and partitions objects into lifecycle stages.
// Hook stages are stably sorted by hook weight in ascending order.
func PlanManifest(manifest []byte) (*ManifestPlan, error) {
	plan := &ManifestPlan{}
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(string(manifest)), 4096)

	for {
		obj := &unstructured.Unstructured{}
		if err := decoder.Decode(obj); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decoding manifest: %w", err)
		}

		if len(obj.Object) == 0 {
			continue // Skip empty documents
		}

		hook, isHook := ParseHook(obj)
		if !isHook {
			plan.Main = append(plan.Main, obj)
			continue
		}

		matchedStage := false

		// Combining Install & Upgrade Events inspired by Argo CD
		// https://argo-cd.readthedocs.io/en/latest/user-guide/helm/#helm-hooks
		if HasHookEvent(hook, helmRelease.HookPreInstall) || HasHookEvent(hook, helmRelease.HookPreUpgrade) {
			plan.PreInstall = append(plan.PreInstall, obj)
			matchedStage = true
		}
		if HasHookEvent(hook, helmRelease.HookPostInstall) || HasHookEvent(hook, helmRelease.HookPostUpgrade) {
			plan.PostInstall = append(plan.PostInstall, obj)
			matchedStage = true
		}
		if HasHookEvent(hook, helmRelease.HookPreDelete) {
			plan.PreDelete = append(plan.PreDelete, obj)
			matchedStage = true
		}
		if HasHookEvent(hook, helmRelease.HookPostDelete) {
			plan.PostDelete = append(plan.PostDelete, obj)
			matchedStage = true
		}

		// If a hook declared other events (e.g., test) but no matched lifecycle stage, keep it out of Main
		_ = matchedStage
	}

	if err := sortStageByWeight(plan.PreInstall); err != nil {
		return nil, fmt.Errorf("sort pre-install hooks: %w", err)
	}
	if err := sortStageByWeight(plan.PostInstall); err != nil {
		return nil, fmt.Errorf("sort post-install hooks: %w", err)
	}
	if err := sortStageByWeight(plan.PreDelete); err != nil {
		return nil, fmt.Errorf("sort pre-delete hooks: %w", err)
	}
	if err := sortStageByWeight(plan.PostDelete); err != nil {
		return nil, fmt.Errorf("sort post-delete hooks: %w", err)
	}

	return plan, nil
}

func sortStageByWeight(objects []*unstructured.Unstructured) error {
	weights := make([]int, len(objects))
	for i, obj := range objects {
		w, err := ParseHookWeight(obj)
		if err != nil {
			return err
		}
		weights[i] = w
	}

	// Pair indices with weights to allow stable sort
	indices := make([]int, len(objects))
	for i := range indices {
		indices[i] = i
	}

	sort.SliceStable(indices, func(i, j int) bool {
		return weights[indices[i]] < weights[indices[j]]
	})

	sorted := make([]*unstructured.Unstructured, len(objects))
	for i, idx := range indices {
		sorted[i] = objects[idx]
	}
	copy(objects, sorted)

	return nil
}
