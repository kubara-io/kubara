package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	helmRelease "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestParseHook(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetName("hook-cm")
	obj.SetKind("ConfigMap")
	obj.SetAnnotations(map[string]string{
		"helm.sh/hook":               "pre-install, post-upgrade",
		"helm.sh/hook-delete-policy": "hook-succeeded, before-hook-creation",
		"helm.sh/hook-weight":        "-10",
	})

	hook, ok := ParseHook(obj)

	require.True(t, ok)
	assert.Equal(t, "hook-cm", hook.Name)
	assert.Equal(t, "ConfigMap", hook.Kind)

	assert.Equal(
		t,
		[]helmRelease.HookEvent{helmRelease.HookPreInstall, helmRelease.HookPostUpgrade},
		hook.Events,
	)

	assert.True(t, HasHookEvent(hook, helmRelease.HookPreInstall))
	assert.True(t, HasHookEvent(hook, helmRelease.HookPostUpgrade))
	assert.False(t, HasHookEvent(hook, helmRelease.HookPreDelete))

	assert.True(
		t,
		HasHookDeletePolicy(hook, helmRelease.HookBeforeHookCreation),
	)

	assert.True(
		t,
		HasHookDeletePolicy(hook, helmRelease.HookSucceeded),
	)

	assert.False(
		t,
		HasHookDeletePolicy(hook, helmRelease.HookFailed),
	)

	assert.Equal(t, -10, hook.Weight)

	parsedWeight, err := ParseHookWeight(obj)
	require.NoError(t, err)
	assert.Equal(t, -10, parsedWeight)
}

func TestParseHookRequiresHookAnnotation(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetKind("Job")
	obj.SetAnnotations(map[string]string{
		"helm.sh/hook-delete-policy": "before-hook-creation",
	})

	_, ok := ParseHook(obj)

	assert.False(t, ok)
}

func TestHookWeightDefaultsToZero(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetAnnotations(map[string]string{
		"helm.sh/hook": "post-install",
	})

	hook, ok := ParseHook(obj)
	require.True(t, ok)
	assert.Equal(t, 0, hook.Weight)

	weight, err := ParseHookWeight(obj)
	require.NoError(t, err)
	assert.Equal(t, 0, weight)
}

func TestHookWeightRejectsInvalidValue(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetAnnotations(map[string]string{
		"helm.sh/hook":        "post-install",
		"helm.sh/hook-weight": "invalid",
	})

	hook, ok := ParseHook(obj)
	require.True(t, ok)
	assert.Equal(t, 0, hook.Weight)

	_, err := ParseHookWeight(obj)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `parse Helm hook weight "invalid"`)
}
