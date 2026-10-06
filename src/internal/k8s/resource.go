package k8s

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
)

func (c *Client) resourceInterface(
	obj *unstructured.Unstructured,
) (dynamic.ResourceInterface, error) {
	gvr, scope, err := c.getGVR(
		obj.GroupVersionKind(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get GVR for %q: %w",
			obj.GroupVersionKind().String(),
			err,
		)
	}

	if scope == meta.RESTScopeNamespace {
		if obj.GetNamespace() == "" {
			obj.SetNamespace("default")
		}

		return c.DynamicClient.
			Resource(gvr).
			Namespace(obj.GetNamespace()), nil
	}

	return c.DynamicClient.Resource(gvr), nil
}

func (c *Client) getGVR(
	gvk schema.GroupVersionKind,
) (schema.GroupVersionResource, meta.RESTScope, error) {
	mapping, err := c.RESTMapper.RESTMapping(
		gvk.GroupKind(),
		gvk.Version,
	)
	if err != nil {
		return schema.GroupVersionResource{}, nil, fmt.Errorf(
			"resolve REST mapping for %q: %w",
			gvk.String(),
			err,
		)
	}

	return mapping.Resource, mapping.Scope, nil
}

// DeleteObjectAndWait deletes exactly the currently existing Kubernetes object.
//
// The UID precondition prevents deleting a replacement object that happens to
// acquire the same namespace/name between GET and DELETE.
func (c *Client) DeleteObjectAndWait(
	ctx context.Context,
	obj *unstructured.Unstructured,
	timeout time.Duration,
) error {
	dr, err := c.resourceInterface(obj)
	if err != nil {
		return err
	}

	existing, err := dr.Get(
		ctx,
		obj.GetName(),
		metav1.GetOptions{},
	)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf(
			"get existing object: %w",
			err,
		)
	}

	uid := existing.GetUID()
	if uid == "" {
		return fmt.Errorf(
			"existing %s %s/%s has no UID",
			obj.GetKind(),
			obj.GetNamespace(),
			obj.GetName(),
		)
	}

	propagation := metav1.DeletePropagationForeground

	err = dr.Delete(
		ctx,
		obj.GetName(),
		metav1.DeleteOptions{
			PropagationPolicy: &propagation,
			Preconditions: &metav1.Preconditions{
				UID: &uid,
			},
		},
	)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf(
			"delete existing object: %w",
			err,
		)
	}

	if timeout <= 0 {
		timeout = time.Minute
	}

	err = wait.PollUntilContextTimeout(
		ctx,
		time.Second,
		timeout,
		true,
		func(ctx context.Context) (bool, error) {
			current, err := dr.Get(
				ctx,
				obj.GetName(),
				metav1.GetOptions{},
			)

			if apierrors.IsNotFound(err) {
				return true, nil
			}
			if err != nil {
				return false, fmt.Errorf(
					"check object deletion: %w",
					err,
				)
			}

			// The original object is gone. A different object now owns
			// the same namespace/name, so our deletion is complete.
			if current.GetUID() != uid {
				return true, nil
			}

			return false, nil
		},
	)
	if err != nil {
		return fmt.Errorf(
			"wait for %s %s/%s deletion: %w",
			obj.GetKind(),
			obj.GetNamespace(),
			obj.GetName(),
			err,
		)
	}

	return nil
}

// WaitForObjectCompletion waits for run-to-completion hook resources.
//
// Jobs and Pods have explicit completion semantics. Other Kubernetes resource
// kinds are considered complete once the API server has accepted their apply.
func (c *Client) WaitForObjectCompletion(
	ctx context.Context,
	obj *unstructured.Unstructured,
	timeout time.Duration,
) error {
	switch obj.GetKind() {
	case "Job", "Pod":
	default:
		return nil
	}

	dr, err := c.resourceInterface(obj)
	if err != nil {
		return err
	}

	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	err = wait.PollUntilContextTimeout(
		ctx,
		time.Second,
		timeout,
		true,
		func(ctx context.Context) (bool, error) {
			current, err := dr.Get(
				ctx,
				obj.GetName(),
				metav1.GetOptions{},
			)

			if apierrors.IsNotFound(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}

			switch obj.GetKind() {
			case "Job":
				return jobCompleted(current)

			case "Pod":
				return podCompleted(current)
			}

			return true, nil
		},
	)
	if err != nil {
		return fmt.Errorf(
			"wait for hook %s %s/%s: %w",
			obj.GetKind(),
			obj.GetNamespace(),
			obj.GetName(),
			err,
		)
	}

	return nil
}

func jobCompleted(
	obj *unstructured.Unstructured,
) (bool, error) {
	conditions, _, err := unstructured.NestedSlice(
		obj.Object,
		"status",
		"conditions",
	)
	if err != nil {
		return false, err
	}

	for _, raw := range conditions {
		condition, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		conditionType, _, _ := unstructured.NestedString(
			condition,
			"type",
		)
		status, _, _ := unstructured.NestedString(
			condition,
			"status",
		)

		if status != "True" {
			continue
		}

		switch conditionType {
		case "Complete":
			return true, nil

		case "Failed":
			reason, _, _ := unstructured.NestedString(
				condition,
				"reason",
			)
			message, _, _ := unstructured.NestedString(
				condition,
				"message",
			)

			return false, fmt.Errorf(
				"job failed: %s: %s",
				reason,
				message,
			)
		}
	}

	return false, nil
}

func podCompleted(
	obj *unstructured.Unstructured,
) (bool, error) {
	phase, _, err := unstructured.NestedString(
		obj.Object,
		"status",
		"phase",
	)
	if err != nil {
		return false, err
	}

	switch phase {
	case "Succeeded":
		return true, nil

	case "Failed":
		reason, _, _ := unstructured.NestedString(
			obj.Object,
			"status",
			"reason",
		)
		message, _, _ := unstructured.NestedString(
			obj.Object,
			"status",
			"message",
		)

		return false, fmt.Errorf(
			"pod failed: %s: %s",
			reason,
			message,
		)

	default:
		return false, nil
	}
}
