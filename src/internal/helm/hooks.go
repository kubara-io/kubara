package helm

import (
	"fmt"
	"strconv"
	"strings"

	helmRelease "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ParseHook parses Helm hook annotations into a helm.sh/helm/v4 Hook object.
func ParseHook(obj *unstructured.Unstructured) (*helmRelease.Hook, bool) {
	annotations := obj.GetAnnotations()

	rawEvents := splitHookAnnotationValues(annotations[helmRelease.HookAnnotation])
	if len(rawEvents) == 0 {
		return nil, false
	}

	events := make([]helmRelease.HookEvent, 0, len(rawEvents))
	for _, event := range rawEvents {
		events = append(events, helmRelease.HookEvent(event))
	}

	rawPolicies := splitHookAnnotationValues(
		annotations[helmRelease.HookDeleteAnnotation],
	)

	policies := make([]helmRelease.HookDeletePolicy, 0, len(rawPolicies))
	for _, policy := range rawPolicies {
		policies = append(policies, helmRelease.HookDeletePolicy(policy))
	}

	weight := 0
	if rawWeight := strings.TrimSpace(annotations[helmRelease.HookWeightAnnotation]); rawWeight != "" {
		parsedWeight, err := strconv.Atoi(rawWeight)
		if err != nil {
			// In case of error, we still return the hook but note:
			// caller can validate or PlanManifest will catch it
			weight = 0
		} else {
			weight = parsedWeight
		}
	}

	return &helmRelease.Hook{
		Name:           obj.GetName(),
		Kind:           obj.GetKind(),
		Events:         events,
		DeletePolicies: policies,
		Weight:         weight,
	}, true
}

// ParseHookWeight parses and validates the hook weight annotation strictly.
func ParseHookWeight(obj *unstructured.Unstructured) (int, error) {
	annotations := obj.GetAnnotations()
	rawWeight := strings.TrimSpace(annotations[helmRelease.HookWeightAnnotation])
	if rawWeight == "" {
		return 0, nil
	}

	weight, err := strconv.Atoi(rawWeight)
	if err != nil {
		return 0, fmt.Errorf("parse Helm hook weight %q: %w", rawWeight, err)
	}

	return weight, nil
}

// HasHookEvent reports whether the hook declares the given event.
func HasHookEvent(hook *helmRelease.Hook, event helmRelease.HookEvent) bool {
	if hook == nil {
		return false
	}
	for _, candidate := range hook.Events {
		if candidate == event {
			return true
		}
	}

	return false
}

// HasHookDeletePolicy reports whether the hook declares the given deletion policy.
func HasHookDeletePolicy(hook *helmRelease.Hook, policy helmRelease.HookDeletePolicy) bool {
	if hook == nil {
		return false
	}
	for _, candidate := range hook.DeletePolicies {
		if candidate == policy {
			return true
		}
	}

	return false
}

func splitHookAnnotationValues(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))

	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}

	return values
}
