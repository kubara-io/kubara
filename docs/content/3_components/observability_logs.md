
# Logs

Centralized log processing in our Kubernetes observability stack is handled by **Loki** and **Alloy**, with **Grafana** as the frontend for analysis. This stack provides a lightweight and scalable logging solution tailored for Kubernetes - similar to what Prometheus offers for metrics.

## Component Overview

- **Loki** is a log aggregation system that indexes logs by labels (e.g., pod, namespace, container). Unlike traditional logging systems (e.g., Elasticsearch), Loki does not index log content, which results in lower resource usage.
- **Alloy** runs as a DaemonSet on each node and collects logs directly from container log files. These logs are then pushed to Loki.
- **Grafana** acts as the user interface for exploring and analyzing logs with filters, queries, and correlations to metrics.

## Access & Integration

Logs can be accessed directly in Grafana under **Explore > Loki**. This interface allows you to search, filter, and group logs by labels, and even combine them with Prometheus metrics.

![Grafana Loki](../images/loki.jpeg)

You can find more about Grafana in the [Dashboards](../3_components/observability_dashboards.md) chapter.

## kubara Standardization

In kubara, **Alloy is enabled by default**. It is configured to automatically collect logs and enrich them with Kubernetes metadata such as namespace, pod name, container, and user-defined labels.

This ensures that logs from all applications are captured consistently and are immediately available for centralized analysis without per-project setup.

## Label-Based Logging

A core concept of Loki is **label-based filtering and grouping**. Typical labels applied automatically include:

- `namespace`
- `pod`
- `container`
- `app` (from pod labels)
- `loglevel` (optionally extracted from log content)

This enables precise queries like:

```
{namespace="my-app", loglevel="error"} |= "failed"
```

## Best Practices

- Use structured logging (e.g., JSON) whenever possible for better field extraction.
- Maintain consistent labels - especially `app`, `loglevel`, `team`, etc.
- Leverage Grafana's "Live Tail" feature for real-time log monitoring.
- Combine logs with metrics for faster root cause analysis (e.g., using "Split View").