package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestDeleteObjectAndWaitMissingObjectSucceeds(t *testing.T) {
	dynamicClient := dynamicfake.NewSimpleDynamicClient(
		runtime.NewScheme(),
	)

	dynamicClient.PrependReactor(
		"get",
		"jobs",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, apierrors.NewNotFound(
				schema.GroupResource{
					Group:    "batch",
					Resource: "jobs",
				},
				"dex-restarter",
			)
		},
	)

	client := &Client{
		DynamicClient: dynamicClient,
		RESTMapper:    testResourceRESTMapper(),
	}

	err := client.DeleteObjectAndWait(
		context.Background(),
		testResourceJob("dex-restarter", ""),
		time.Second,
	)

	require.NoError(t, err)

	for _, action := range dynamicClient.Actions() {
		assert.NotEqual(t, "delete", action.GetVerb())
	}
}

func TestDeleteObjectAndWaitUsesUIDPrecondition(t *testing.T) {
	const name = "dex-restarter"

	oldUID := types.UID("old-job-uid")

	existing := testResourceJob(
		name,
		oldUID,
	)

	dynamicClient := dynamicfake.NewSimpleDynamicClient(
		runtime.NewScheme(),
	)

	getCalls := 0

	dynamicClient.PrependReactor(
		"get",
		"jobs",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			getCalls++

			if getCalls == 1 {
				return true, existing.DeepCopy(), nil
			}

			return true, nil, apierrors.NewNotFound(
				schema.GroupResource{
					Group:    "batch",
					Resource: "jobs",
				},
				name,
			)
		},
	)

	dynamicClient.PrependReactor(
		"delete",
		"jobs",
		func(action k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, nil
		},
	)

	client := &Client{
		DynamicClient: dynamicClient,
		RESTMapper:    testResourceRESTMapper(),
	}

	err := client.DeleteObjectAndWait(
		context.Background(),
		testResourceJob(name, ""),
		time.Second,
	)
	require.NoError(t, err)

	var deleteAction k8stesting.DeleteAction

	for _, action := range dynamicClient.Actions() {
		if action.GetVerb() == "delete" {
			var ok bool

			deleteAction, ok = action.(k8stesting.DeleteAction)
			require.True(t, ok)

			break
		}
	}

	require.NotNil(t, deleteAction)

	deleteOptions := deleteAction.GetDeleteOptions()

	require.NotNil(t, deleteOptions.Preconditions)
	require.NotNil(t, deleteOptions.Preconditions.UID)

	assert.Equal(
		t,
		oldUID,
		*deleteOptions.Preconditions.UID,
	)
}

func TestJobCompleted(t *testing.T) {
	job := testResourceJob("hook-job", "")

	job.Object["status"] = map[string]any{
		"conditions": []any{
			map[string]any{
				"type":   "Complete",
				"status": "True",
			},
		},
	}

	complete, err := jobCompleted(job)

	require.NoError(t, err)
	assert.True(t, complete)
}

func TestJobFailed(t *testing.T) {
	job := testResourceJob("hook-job", "")

	job.Object["status"] = map[string]any{
		"conditions": []any{
			map[string]any{
				"type":    "Failed",
				"status":  "True",
				"reason":  "BackoffLimitExceeded",
				"message": "job failed",
			},
		},
	}

	complete, err := jobCompleted(job)

	require.Error(t, err)
	assert.False(t, complete)
	assert.Contains(
		t,
		err.Error(),
		"BackoffLimitExceeded",
	)
}

func testResourceJob(
	name string,
	uid types.UID,
) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}

	obj.SetGroupVersionKind(
		schema.GroupVersionKind{
			Group:   "batch",
			Version: "v1",
			Kind:    "Job",
		},
	)

	obj.SetName(name)
	obj.SetNamespace("argocd")
	obj.SetUID(uid)

	return obj
}

func testResourceRESTMapper() meta.RESTMapper {
	mapper := meta.NewDefaultRESTMapper(
		[]schema.GroupVersion{
			{
				Group:   "batch",
				Version: "v1",
			},
		},
	)

	mapper.AddSpecific(
		schema.GroupVersionKind{
			Group:   "batch",
			Version: "v1",
			Kind:    "Job",
		},
		schema.GroupVersionResource{
			Group:    "batch",
			Version:  "v1",
			Resource: "jobs",
		},
		schema.GroupVersionResource{
			Group:    "batch",
			Version:  "v1",
			Resource: "job",
		},
		meta.RESTScopeNamespace,
	)

	return mapper
}
