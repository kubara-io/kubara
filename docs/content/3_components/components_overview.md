# Component Overview

This document provides an overview of the tools included in the kubara framework, along with their functionality and key features.
More tools will be added in future releases of the kubara framework.

---

## 1. Application Management

| Tool                                                                  | Description                                                                                          | Functionality                    | Key Features                                                                                 |
|-----------------------------------------------------------------------| ---------------------------------------------------------------------------------------------------- | -------------------------------- | -------------------------------------------------------------------------------------------- |
| <div style="width: 80px;">![Argo CD](../images/argocd-logo.png)</div> | Argo CD. GitOps-based tool for continuous deployment and synchronization of Kubernetes applications. | GitOps-based deployment and sync | - Git integration<br>- Rollbacks<br>- Real-time status monitoring<br>- Multi-cluster support |
| <div style="width: 80px;">![Homer](../images/homer-dashboard-logo.png)</div> | Homer. Simple static dashboard to manage service links via YAML.                                     | Static link collection           | - Grouped links<br>- Easy configuration<br>- Quick navigation                                |

---

## 2. Observability

| Tool                                                                           | Description                                                                                        | Functionality               | Key Features                                                                                            |
|--------------------------------------------------------------------------------| -------------------------------------------------------------------------------------------------- | --------------------------- | ------------------------------------------------------------------------------------------------------- |
| <div style="width: 80px;">![Prometheus](../images/prometheus-logo.png)</div>   | Kube-Prometheus-Stack. Monitoring and alerting toolkit using Prometheus, Grafana, and Alertmanager. | Monitoring for Kubernetes   | - Prometheus metrics<br>- Grafana dashboards<br>- Alertmanager notifications<br>- Pre-configured alerts |
| <div style="width: 80px;">![Grafana Loki](../images/grafana_loki-logo.png)</div>  | Grafana Loki. Log aggregation system for Kubernetes logs. | Log collection and analysis | - Grafana integration<br>- Label-based filtering<br>- Efficient log storage<br>- Scalable architecture  |
| <div style="width: 80px;">![Metrics Server](../images/metrics-server-logo.png)</div> | Metrics Server. Collects resource metrics from Kubernetes nodes and pods. | Resource metric collection  | - Integrates with Horizontal Pod Autoscaler<br>- Lightweight<br>- Kubelet-based collection              |

---

## 3. Security

| Tool                                                                                                                                                                  | Description                                                                     | Functionality              | Key Features                                                                        |
|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------| ------------------------------------------------------------------------------- | -------------------------- | ----------------------------------------------------------------------------------- |
| <div style="width: 80px;">![Cert Manager](../images/cert-manager-logo.png)</div>                                                                                      | Cert Manager. Automates TLS certificate creation and management.                | TLS certificate automation | - ACME support<br>- Auto renewal<br>- Ingress integration                           |
| <div style="width: 80px;">![External Secrets](../images/external-secrets-logo.png)</div> | External Secrets Operator. Sync secrets from external backends into Kubernetes. | Secret synchronization     | - Vault, AWS, GCP support<br>- Auto updates<br>- Encryption                         |
| <div style="width: 80px;">![Kyverno](../images/kyverno-logo.png)</div> | Kyverno. Kubernetes-native policy engine for governance and security.           | Policy management          | - Validation and mutation<br>- Custom policies<br>- GitOps friendly                 |
| <div style="width: 80px;">![OAuth Proxy](../images/oauth-proxy-logo.png) </div>  | OAuth2 proxy for authenticating web applications.                               | Auth via OAuth2/OIDC       | - Google, GitHub, OIDC support<br>- Easy integration<br>- Access control via tokens |

---

## 4. Storage

| Tool                                                            | Description                                                | Functionality      | Key Features                                                        |
| --------------------------------------------------------------- | ---------------------------------------------------------- | ------------------ | ------------------------------------------------------------------- |
| <div style="width: 80px;">![Longhorn](../images/longhorn-logo.png)</div> | Longhorn. Distributed block storage system for Kubernetes. | Persistent storage | - Replication<br>- Snapshots<br>- Backups<br>- Dynamic provisioning |

---

## 5. Network

| Tool                                                                                                                                                     | Description                                                                                                        | Functionality                  | Key Features                                                                            |
|----------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------| ------------------------------ | --------------------------------------------------------------------------------------- |
| <div style="width: 80px;">![External DNS](../images/external-dns-logo.png)</div> | External DNS. Sync DNS records from Kubernetes to external DNS providers.                                          | DNS automation                 | - AWS Route53, Google DNS support<br>- Auto DNS updates<br>- Label-based mapping        |
| <div style="width: 80px;">![Ingress NGINX](../images/ingress-nginx-logo.png)</div> | NGINX Ingress Controller. Ingress controller for HTTP/HTTPS routing in Kubernetes. | Web traffic routing            | - TLS support<br>- Path/host-based routing<br>- Annotations for custom rules            |
| <div style="width: 80px;">![MetalLB](../images/metallb-logo.png) </div>  | MetallLB. Load balancer for bare-metal Kubernetes clusters.                                                        | Load balancing                 | - Layer 2 and BGP modes<br>- IP address pool management<br>- Simple configuration       |

---

## 6. CI/CD

| Tool                                                                                                                                        | Description                                                       | Functionality       | Key Features                                             |
|---------------------------------------------------------------------------------------------------------------------------------------------| ----------------------------------------------------------------- | ------------------- | -------------------------------------------------------- |
| <div style="width: 80px;">![Forgejo](../images/forgejo-logo.png) </div> | Forgejo. Managed Git service with CI/CD integration from STACKIT. | Git repo management | - Web UI<br>- User management<br>- Repos <br>- Pipelines |

---

## Custom Resource Dependencies

If you deactivate or replace applications (Y-axis) with others not part of the kubara framework, be sure to resolve custom resource dependencies such as ServiceMonitors, Certificates, and Secrets accordingly.

| ↓                       | argo-cd | homer-dashboard | kube-prometheus-stack | loki | metrics-server | cert-manager | external-secrets | kyverno | kyverno-policies | kyverno-policy-reporter | oauth2-proxy | longhorn | external-dns | ingress-nginx | metallb |
| ----------------------- | ------- | --------------- | --------------------- | ---- | -------------- | ------------ | ---------------- | ------- | ---------------- | ----------------------- | ------------ | -------- | ------------ | ------------- | ------- |
| argo-cd                 |         |                 |                       |      |                |              |                  |         |                  |                         |              |          |              |               |         |
| homer-dashboard         |         |                 |                       |      |                |              |                  |         |                  |                         |              |          |              |               |         |
| kube-prometheus-stack   | X       |                 |                       |      |                | X            | X                | X       |                  |                         |              |          | X            | X             |         |
| loki                    |         |                 |                       |      |                |              |                  |         |                  |                         |              |          |              |               |         |
| metric-server           |         |                 |                       |      |                |              |                  |         |                  |                         |              |          |              |               |         |
| cert-manager            | X       |                 | X                     | X    |                |              |                  |         |                  |                         |              |          |              |               |         |
| external-secrets        | X       |                 | X                     | X    |                | X            |                  |         |                  |                         |              |          | X            |               |         |
| kyverno                 |         |                 |                       |      |                |              |                  |         |                  | X                       |              |          |              |               |         |
| kyverno-policies        |         |                 |                       |      |                |              |                  |         |                  | X                       |              |          |              |               |         |
| kyverno-policy-reporter |         |                 |                       |      |                |              |                  |         |                  |                         |              |          |              |               |         |
| oauth2-proxy            | X       | X               | X                     |      |                |              |                  |         |                  | X                       |              |          |              |               |         |
| longhorn                | X       |                 | X                     | X    |                |              |                  |         |                  |                         |              |          |              |               |         |
| external-dns            |         |                 |                       |      |                |              |                  |         |                  |                         |              |          |              |               |         |
| ingress-nginx           |         |                 |                       |      |                |              |                  |         |                  |                         |              |          |              |               |         |
| metalLB                 | X       | X               | X                     |      |                |              |                  |         |                  | X                       |              |          |              | X             |         |

