package helm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kubara-io/kubara/internal/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	release "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type fakeHookClient struct {
	actions []string
}

func (f *fakeHookClient) ApplyManifest(
	_ context.Context,
	manifest []byte,
	_ k8s.ApplyOptions,
) error {
	f.actions = append(
		f.actions,
		"apply:"+manifestName(manifest),
	)

	return nil
}

func (f *fakeHookClient) DeleteObjectAndWait(
	_ context.Context,
	obj *unstructured.Unstructured,
	_ time.Duration,
) error {
	f.actions = append(
		f.actions,
		"delete:"+obj.GetName(),
	)

	return nil
}

func (f *fakeHookClient) WaitForObjectCompletion(
	_ context.Context,
	obj *unstructured.Unstructured,
	_ time.Duration,
) error {
	f.actions = append(
		f.actions,
		"wait:"+obj.GetName(),
	)

	return nil
}

func TestApplyWithHooksRunsPhasesInOrder(t *testing.T) {
	manifest := []byte(`
apiVersion: batch/v1
kind: Job
metadata:
  name: pre
  annotations:
    helm.sh/hook: pre-install
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: normal
---
apiVersion: batch/v1
kind: Job
metadata:
  name: post
  annotations:
    helm.sh/hook: post-install
`)

	client := &fakeHookClient{}

	hooks := NewBootstrapHooks(
		client,
		BootstrapHookOptions{},
	)

	err := ApplyWithHooks(
		context.Background(),
		manifest,
		hooks,
		func(
			_ context.Context,
			resources []byte,
		) error {
			client.actions = append(
				client.actions,
				"apply:resources",
			)

			assert.Contains(
				t,
				string(resources),
				"name: normal",
			)

			return nil
		},
		func(_ context.Context) error {
			client.actions = append(
				client.actions,
				"ready",
			)

			return nil
		},
	)
	require.NoError(t, err)

	assert.Equal(
		t,
		[]string{
			"delete:pre",
			"apply:pre",
			"wait:pre",
			"apply:resources",
			"ready",
			"delete:post",
			"apply:post",
			"wait:post",
		},
		client.actions,
	)
}

func TestCombinedPreInstallAndUpgradeRunsOnce(t *testing.T) {
	hook := &release.Hook{
		Name: "combined",
		Kind: "Job",
		Events: []release.HookEvent{
			release.HookPreInstall,
			release.HookPreUpgrade,
		},
		Manifest: `
apiVersion: batch/v1
kind: Job
metadata:
  name: combined
`,
	}

	client := &fakeHookClient{}

	hooks := NewBootstrapHooks(
		client,
		BootstrapHookOptions{},
	)

	require.NoError(
		t,
		hooks.PreApply(
			context.Background(),
			[]*release.Hook{
				hook,
			},
		),
	)

	assert.Equal(
		t,
		[]string{
			"delete:combined",
			"apply:combined",
			"wait:combined",
		},
		client.actions,
	)
}

func TestHooksExecuteByWeight(t *testing.T) {
	hooksToRun := []*release.Hook{
		testHook("late", 10, release.HookPreInstall),
		testHook("early", -10, release.HookPreInstall),
		testHook("middle", 0, release.HookPreInstall),
	}

	client := &fakeHookClient{}

	hooks := NewBootstrapHooks(
		client,
		BootstrapHookOptions{},
	)

	require.NoError(
		t,
		hooks.PreApply(
			context.Background(),
			hooksToRun,
		),
	)

	assert.Equal(
		t,
		[]string{
			"delete:early",
			"apply:early",
			"wait:early",
			"delete:middle",
			"apply:middle",
			"wait:middle",
			"delete:late",
			"apply:late",
			"wait:late",
		},
		client.actions,
	)
}

func TestDeleteWithHooksRunsDeletePhases(t *testing.T) {
	manifest := []byte(`
apiVersion: batch/v1
kind: Job
metadata:
  name: pre-delete
  annotations:
    helm.sh/hook: pre-delete
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: normal
---
apiVersion: batch/v1
kind: Job
metadata:
  name: post-delete
  annotations:
    helm.sh/hook: post-delete
`)

	client := &fakeHookClient{}

	hooks := NewBootstrapHooks(
		client,
		BootstrapHookOptions{},
	)

	err := DeleteWithHooks(
		context.Background(),
		manifest,
		hooks,
		func(
			_ context.Context,
			_ []byte,
		) error {
			client.actions = append(
				client.actions,
				"delete:resources",
			)

			return nil
		},
	)
	require.NoError(t, err)

	assert.Equal(
		t,
		[]string{
			"delete:pre-delete",
			"apply:pre-delete",
			"wait:pre-delete",
			"delete:resources",
			"delete:post-delete",
			"apply:post-delete",
			"wait:post-delete",
		},
		client.actions,
	)
}

func TestHookSucceededCleanup(t *testing.T) {
	hook := testHook(
		"cleanup",
		0,
		release.HookPreInstall,
	)

	hook.DeletePolicies = []release.HookDeletePolicy{
		release.HookBeforeHookCreation,
		release.HookSucceeded,
	}

	client := &fakeHookClient{}

	hooks := NewBootstrapHooks(
		client,
		BootstrapHookOptions{},
	)

	require.NoError(
		t,
		hooks.PreApply(
			context.Background(),
			[]*release.Hook{
				hook,
			},
		),
	)

	assert.Equal(
		t,
		[]string{
			"delete:cleanup",
			"apply:cleanup",
			"wait:cleanup",
			"delete:cleanup",
		},
		client.actions,
	)
}

func TestProtectedHooksAreNotDeleted(t *testing.T) {
	tests := []struct {
		name string
		kind string
	}{
		{
			name: "crd",
			kind: "CustomResourceDefinition",
		},
		{
			name: "namespace",
			kind: "Namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := &release.Hook{
				Name: tt.name,
				Kind: tt.kind,
				DeletePolicies: []release.HookDeletePolicy{
					release.HookBeforeHookCreation,
				},
				Manifest: `
apiVersion: v1
kind: ` + tt.kind + `
metadata:
  name: ` + tt.name + `
`,
			}

			client := &fakeHookClient{}

			hooks := NewBootstrapHooks(
				client,
				BootstrapHookOptions{},
			)

			require.NoError(
				t,
				hooks.DeleteByPolicy(
					context.Background(),
					hook,
					release.HookBeforeHookCreation,
				),
			)

			assert.Empty(
				t,
				client.actions,
			)
		})
	}
}

func TestParseManifestSetRejectsUnsupportedHookEvent(t *testing.T) {
	manifest := []byte(`
apiVersion: batch/v1
kind: Job
metadata:
  name: test-hook
  annotations:
    helm.sh/hook: test
`)

	_, err := ParseManifestSet(manifest)

	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		`unsupported Helm bootstrap hook event "test"`,
	)
}

func TestParseManifestSetRejectsMixedHookAnnotationsOnResource(t *testing.T) {
	manifest := []byte(`
apiVersion: batch/v1
kind: Job
metadata:
  name: mixed
  annotations:
    helm.sh/hook: post-install
    argocd.argoproj.io/hook: PostSync
`)

	_, err := ParseManifestSet(manifest)

	require.Error(t, err)
	assert.Contains(
		t,
		err.Error(),
		"declares both Helm and Argo CD hook annotations",
	)
}

func TestNormalResourcesKeepTheirOrder(t *testing.T) {
	manifest := []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: first
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: second
`)

	set, err := ParseManifestSet(manifest)
	require.NoError(t, err)

	resources := string(set.Resources)

	assert.Less(
		t,
		strings.Index(resources, "name: first"),
		strings.Index(resources, "name: second"),
	)
}

func testHook(
	name string,
	weight int,
	event release.HookEvent,
) *release.Hook {
	return &release.Hook{
		Name:   name,
		Kind:   "Job",
		Weight: weight,
		Events: []release.HookEvent{
			event,
		},
		Manifest: `
apiVersion: batch/v1
kind: Job
metadata:
  name: ` + name + `
`,
	}
}

func manifestName(
	manifest []byte,
) string {
	decoder := yaml.NewYAMLOrJSONDecoder(
		strings.NewReader(string(manifest)),
		4096,
	)

	obj := &unstructured.Unstructured{}
	if err := decoder.Decode(obj); err != nil {
		return ""
	}

	return obj.GetName()
}
