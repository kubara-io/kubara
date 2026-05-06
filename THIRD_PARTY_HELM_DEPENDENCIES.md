# Third-Party Helm Dependencies

kubara includes umbrella Helm charts under `src/internal/catalog/built-in/managed-service-catalog/helm`.
When kubara is used, Helm resolves these third-party chart dependencies from the configured upstream repositories
(`helm dependency update`) and they may be rendered or deployed into the target cluster.

kubara release archives do not bundle these upstream chart packages by default.
Each dependency remains licensed by its upstream project.

## External chart dependencies

| Upstream dependency | Upstream GitHub repository | License |
| --- | --- | --- |
| `argo-cd` | [argoproj/argo-helm](https://github.com/argoproj/argo-helm) | `Apache-2.0` |
| `cert-manager` | [cert-manager/cert-manager](https://github.com/cert-manager/cert-manager) | `Apache-2.0` |
| `external-dns` | [kubernetes-sigs/external-dns](https://github.com/kubernetes-sigs/external-dns) | `Apache-2.0` |
| `external-secrets` | [external-secrets/external-secrets](https://github.com/external-secrets/external-secrets) | `Apache-2.0` |
| `kube-prometheus-stack` | [prometheus-community/helm-charts](https://github.com/prometheus-community/helm-charts) | `Apache-2.0` |
| `prometheus-blackbox-exporter` | [prometheus-community/helm-charts](https://github.com/prometheus-community/helm-charts) | `Apache-2.0` |
| `kyverno` | [kyverno/kyverno](https://github.com/kyverno/kyverno) | `Apache-2.0` |
| `kyverno-policies` | [kyverno/kyverno](https://github.com/kyverno/kyverno) | `Apache-2.0` |
| `policy-reporter` | [kyverno/policy-reporter](https://github.com/kyverno/policy-reporter) | `MIT` |
| `loki` | [grafana/helm-charts](https://github.com/grafana/helm-charts) | `Apache-2.0` |
| `alloy` | [grafana/helm-charts](https://github.com/grafana/helm-charts) | `Apache-2.0` |
| `longhorn` | [longhorn/longhorn](https://github.com/longhorn/longhorn) | `Apache-2.0` |
| `metallb` | [metallb/metallb](https://github.com/metallb/metallb) | `Apache-2.0` |
| `metrics-server` | [kubernetes-sigs/metrics-server](https://github.com/kubernetes-sigs/metrics-server) | `Apache-2.0` |
| `oauth2-proxy` | [oauth2-proxy/manifests](https://github.com/oauth2-proxy/manifests) | `Apache-2.0` |
| `traefik` | [traefik/traefik-helm-chart](https://github.com/traefik/traefik-helm-chart) | `Apache-2.0` |

Internal `template-library` dependencies (`file://../template-library`) are not listed here.

License values are based on upstream repository SPDX identifiers.
