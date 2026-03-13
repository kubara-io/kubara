
# Metrics

Metrics are the foundation of effective monitoring in Kubernetes. They allow continuous observation and data-driven evaluation of the health and performance of clusters, nodes, pods, and applications.

To collect, store, and visualize metrics, we use the **kube-prometheus-stack**, which includes **Prometheus** along with key exporters like **Node Exporter** and **kube-state-metrics**. The collected data is analyzed and visualized through **Grafana** and can also be explored directly in the **Prometheus UI**.

## Architecture Overview

- **Prometheus** scrapes metrics from defined targets (e.g., nodes, pods, services).
- **Node Exporter** provides system-level metrics (CPU, memory, disk I/O, etc.).
- **kube-state-metrics** exposes metrics about the state of Kubernetes resources (e.g., Deployments, CronJobs, StatefulSets).
- **ServiceMonitors** and **PodMonitors** define which services Prometheus should scrape.
- Configuration is managed declaratively using Helm charts and GitOps (via ArgoCD).

## Accessing Prometheus

The **Prometheus UI** is accessible via Ingress at:

```
https://<customer-domain>/prometheus
```

It provides a functional interface to explore metrics, debug scraping targets, and manually execute PromQL queries.

Note: While Prometheus is excellent for direct queries and troubleshooting, **Grafana** is used as the primary interface for metric visualization, offering rich dashboards and user-friendly analytics.
See more in the [Dashboards](../3_components/observability_dashboards.md) section.



## kubara Standardization

In kubara, `ServiceMonitors` are enabled by default for all deployed applications. This ensures that each app exposes Prometheus-compatible metrics and is automatically included in centralized monitoring.

We also apply consistent labels to every ServiceMonitor-for example, `monitoring.instance`-to simplify filtering and organization.

Example snippet from the ArgoCD Helm chart `values.yaml`:

```yaml
controller:
  metrics:
    enabled: true
    serviceMonitor:
      enabled: true
      additionalLabels:
        monitoring.instance: default
```

## Example: ServiceMonitor

A `ServiceMonitor` defines which services Prometheus should monitor. Here's a basic example:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: my-app
  labels:
    release: kube-prometheus-stack
spec:
  selector:
    matchLabels:
      app: my-app
  namespaceSelector:
    matchNames:
      - my-app-namespace
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
```

## Configuration via `values.yaml`

Prometheus settings are defined in the `values.yaml` file of the Helm chart. This includes scrape intervals, retention policies, and storage settings.

```yaml
prometheus:
  prometheusSpec:
    scrapeInterval: "30s"
    evaluationInterval: "30s"
    retention: "15d"
    storageSpec:
      volumeClaimTemplate:
        spec:
          storageClassName: "gp2"
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 50Gi
```

## Prebuilt Dashboards

The kube-prometheus-stack comes with a wide set of **preconfigured Grafana dashboards** for:

- Kubernetes nodes and workloads
- Prometheus internals
- etcd, API server, scheduler
- kubelet performance
- resource usage and capacity planning

These dashboards are automatically imported when Grafana is deployed with the stack.
You can find more about them in the [Dashboards](../3_components/observability_dashboards.md) chapter.

## Best Practices

- Use `ServiceMonitors` instead of static target definitions to keep deployments flexible and declarative.
- Apply labels for better metric organization and filtering (e.g., by namespace, app, or team).
- Set retention periods based on operational needs-long retention can impact performance.
- Monitor Prometheus itself: metrics like `prometheus_tsdb_head_series` and `prometheus_engine_query_duration_seconds` provide insight into system health and scaling requirements.
