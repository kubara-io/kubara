package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WaitForPod waits for a pod to be ready based on label selector
func (c *Client) WaitForPod(ctx context.Context, namespace, labelSelector string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	var lastErr error

	checkReady := func() (bool, error) {
		podList, err := c.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return false, fmt.Errorf("list pods in namespace %q with selector %q: %w", namespace, labelSelector, err)
		}

		for i := range podList.Items {
			if isPodReady(&podList.Items[i]) {
				return true, nil
			}
		}

		return false, nil
	}

	// Fast-path in case the pod is already ready when the wait starts.
	if ready, err := checkReady(); err != nil {
		lastErr = err
	} else if ready {
		return nil
	}

	for {
		select {
		case <-ticker.C:
			ready, err := checkReady()
			if err != nil {
				lastErr = err
				continue
			}
			if ready {
				return nil
			}
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("timeout waiting for pod with selector %q to be ready after last list error %v: %w", labelSelector, lastErr, ctx.Err())
			}
			return fmt.Errorf("timeout waiting for pod with selector %q to be ready: %w", labelSelector, ctx.Err())
		}
	}
}

// WaitForDeployment waits for a deployment to be ready
func (c *Client) WaitForDeployment(ctx context.Context, namespace, name string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for deployment \"%s/%s\": %w", namespace, name, ctx.Err())

		case <-ticker.C:
			deployment, err := c.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				continue
			}

			// Handle default replicas if nil (Kubernetes defaults to 1)
			desiredReplicas := int32(1)
			if deployment.Spec.Replicas != nil {
				desiredReplicas = *deployment.Spec.Replicas
			}

			// Check readiness
			// 1. Generation check ensures the controller has seen the latest change
			// 2. UpdatedReplicas check ensures no old pods are counted
			// 3. AvailableReplicas check ensures pods are actually ready
			if deployment.Generation <= deployment.Status.ObservedGeneration &&
				deployment.Status.UpdatedReplicas == desiredReplicas &&
				deployment.Status.AvailableReplicas == desiredReplicas {
				return nil
			}
		}
	}
}

// isPodReady checks if a pod is ready
func isPodReady(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}
