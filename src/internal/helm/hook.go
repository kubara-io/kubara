package helm

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	hookAnnotation             = "helm.sh/hook"
	hookDeletePolicyAnnotation = "helm.sh/hook-delete-policy"
	hookWeightAnnotation       = "helm.sh/hook-weight"
)

type HookDeletePolicy string

const (
	HookDeletePolicyBeforeHookCreation HookDeletePolicy = "before-hook-creation"
	HookDeletePolicySucceeded          HookDeletePolicy = "hook-succeeded"
	HookDeletePolicyFailed             HookDeletePolicy = "hook-failed"
)

type Hook struct {
	Events         []string
	DeletePolicies []HookDeletePolicy
	weight         string
}

func ParseHook(obj *unstructured.Unstructured) (Hook, bool) {
	annotations := obj.GetAnnotations()

	events := splitHookAnnotationValues(annotations[hookAnnotation])
	if len(events) == 0 {
		return Hook{}, false
	}

	rawPolicies := splitHookAnnotationValues(
		annotations[hookDeletePolicyAnnotation],
	)

	policies := make([]HookDeletePolicy, 0, len(rawPolicies))
	for _, policy := range rawPolicies {
		policies = append(policies, HookDeletePolicy(policy))
	}

	return Hook{
		Events:         events,
		DeletePolicies: policies,
		weight:         strings.TrimSpace(annotations[hookWeightAnnotation]),
	}, true
}

// HasDeletePolicy reports whether the hook declares the given deletion policy.
func (h Hook) HasDeletePolicy(policy HookDeletePolicy) bool {
	for _, candidate := range h.DeletePolicies {
		if candidate == policy {
			return true
		}
	}

	return false
}

func (h Hook) Weight() (int, error) {
	if h.weight == "" {
		return 0, nil
	}

	weight, err := strconv.Atoi(h.weight)
	if err != nil {
		return 0, fmt.Errorf("parse Helm hook weight %q: %w", h.weight, err)
	}

	return weight, nil
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
