package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestParseHook(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetKind("ConfigMap")
	obj.SetAnnotations(map[string]string{
		"helm.sh/hook":               "pre-install, post-upgrade",
		"helm.sh/hook-delete-policy": "hook-succeeded, before-hook-creation",
		"helm.sh/hook-weight":        "-10",
	})

	hook, ok := ParseHook(obj)

	require.True(t, ok)

	assert.Equal(
		t,
		[]string{"pre-install", "post-upgrade"},
		hook.Events,
	)

	assert.True(
		t,
		hook.HasDeletePolicy(HookDeletePolicyBeforeHookCreation),
	)

	assert.True(
		t,
		hook.HasDeletePolicy(HookDeletePolicySucceeded),
	)

	assert.False(
		t,
		hook.HasDeletePolicy(HookDeletePolicyFailed),
	)

	weight, err := hook.Weight()
	require.NoError(t, err)
	assert.Equal(t, -10, weight)
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

	weight, err := hook.Weight()

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

	_, err := hook.Weight()

	require.Error(t, err)
	assert.Contains(t, err.Error(), `parse Helm hook weight "invalid"`)
}
