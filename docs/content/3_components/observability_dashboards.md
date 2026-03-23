
# Dashboards (Grafana)

**Grafana** is the central user interface for visualizing all collected metrics in our Kubernetes observability stack.  
It allows users not only to query metrics, but also to present them clearly in graphical form - using dashboards, charts, heatmaps, and tables.

## Architecture Overview

Grafana is installed as part of the **kube-prometheus-stack** Helm chart and is accessible via Ingress at:

```
https://<customer-domain>/grafana
```

Authentication is enabled by default to protect access. Users can sign in either via local login or an integrated identity provider (e.g., OAuth2, SSO).

![Grafana Login](../images/grafana-login.jpeg)


## Prebuilt Dashboards

The kube-prometheus-stack includes a wide set of **prebuilt dashboards** that are automatically imported into Grafana:

- Cluster and node overviews
- Pod and container metrics
- Workload performance (Deployments, StatefulSets, DaemonSets)
- Kubernetes control plane components (API server, Scheduler, Controller Manager)
- Prometheus and etcd internals

These dashboards provide an immediate, comprehensive view of key operational metrics with zero setup.

![Grafana Login](../images/grafana-dashboards.jpeg)

![Grafana Login](../images/grafana-metrics1.jpeg)

![Grafana Login](../images/grafana-metrics2.jpeg)

## Logs in Grafana

In Grafana you can also view logs, see [Logs](../3_components/observability_logs.md)

## Alerts in Grafana

In Grafana you can also view alerts, see [alerting](../3_components/observability_alerts.md)

## Custom Dashboards

In addition to the default dashboards, you can create custom dashboards tailored to specific use cases such as:

- Business-specific application metrics
- Project- or team-based views
- Alert overviews and status panels

Dashboards can be created manually through the Grafana UI or defined as JSON files and managed via GitOps.

!!! note
    Custom dashboards can be included via `dashboards.yaml` in the kube-prometheus-stack Helm chart for automated provisioning.

## Dashboard Management

Grafana allows you to:

- Track version history (Save History)
- Export and import dashboards (JSON)
- Organize dashboards into folders
- Set access control (ACL) permissions

Tags can also be applied to categorize and filter dashboards more easily.

## Best Practices

- Use dashboard variables (templating) to provide flexible filtering (e.g., by namespace, cluster, application).
- Version critical dashboards using GitOps.
- Use consistent naming and folder structure.
- Avoid overly complex panels to improve dashboard loading performance.
