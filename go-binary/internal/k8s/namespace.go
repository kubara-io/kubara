package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnsureNamespace creates a namespace if it doesn't exist
func (c *Client) EnsureNamespace(ctx context.Context, name string, dryRun bool) error {
	_, err := c.Clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		// Namespace already exists
		return nil
	}

	// Create namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	opts := metav1.CreateOptions{}
	if dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
	}

	_, err = c.Clientset.CoreV1().Namespaces().Create(ctx, namespace, opts)
	if err != nil {
		return fmt.Errorf("create namespace %q: %w", name, err)
	}

	return nil
}

// ListNamespaces returns all namespaces
func (c *Client) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	namespaces, err := c.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	return namespaces.Items, nil
}
