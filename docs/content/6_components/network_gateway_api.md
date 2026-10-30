# Network: Gateway API (Traefik)

## What is the Kubernetes Gateway API?

The [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/) is the next-generation standard for service networking in Kubernetes, superseding the `Ingress` resource. It introduces role-oriented resources (`GatewayClass`, `Gateway`, `HTTPRoute`, `GRPCRoute`, etc.) that cleanly separate infrastructure operations (defining listening ports and TLS certificates) from application routing (defining paths, headers, and target services).

kubara supports Gateway API alongside traditional Ingress. When configured with Traefik, Traefik functions as both the Gateway controller and Ingress controller simultaneously.

## Architecture & Dual-Provider Operation

When Gateway API is enabled:

1. **Traefik runs both providers in parallel:**
   * `providers.kubernetesGateway`: Watches and reconciles `Gateway` and `HTTPRoute` resources.
   * `providers.kubernetesIngress` & `providers.kubernetesCRD`: Remain active so non-migrated endpoints (such as `cert-manager` HTTP-01 challenge ingresses, OpenBao ingress, and Traefik dashboard `IngressRoute` at `/traefik`) continue to function without interruption.

2. **Automated TLS provisioning:**
   * Traefik provisions a cert-manager `Certificate` (`traefik-gateway-tls`) for the Gateway's HTTPS listener.
   * The listener references `traefik-gateway-tls` in `certificateRefs`, which is also set as Traefik's `tlsStore.default.defaultCertificate`.

3. **Cross-Namespace Route Attachment:**
   * The HTTPS listener is configured with `allowedRoutes.namespaces.from: All`, enabling `HTTPRoute` resources across any namespace (`argocd`, `traefik`, monitoring, or application workloads) to attach to the central Gateway.

4. **Namespace-Local ForwardAuth Middlewares:**
   * In Gateway API, `HTTPRoute` filter references (`ExtensionRef`) are restricted to local object references (`LocalObjectReference`) and cannot span across namespaces.
   * To support OAuth2 Proxy authentication, kubara renders a namespace-local `<release>-gateway-auth` `Middleware` into each service's namespace (and `argocd-gateway-auth` in `argocd`), referencing the central OAuth2 Proxy service.

5. **Automatic DNS Synchronization:**
   * ExternalDNS is automatically configured with `--source=gateway-httproute`, inspecting hostnames defined on `HTTPRoute` resources and syncing DNS records with your cloud provider.

---

## Configuration in `config.yaml`

To enable Gateway API on your cluster, configure the `.cluster.networking` block in `config.yaml`:

```yaml
clusters:
  - name: my-cluster
    stage: dev
    dnsName: my-cluster.example.com

    networking:
      type: gateway # selects Gateway API ("ingress" or "gateway")
      gateway:
        name: traefik # name of the Gateway resource
        namespace: traefik # namespace where Traefik is deployed
        sectionName: websecure # listener section (defaults to websecure)

    services:
      traefik:
        status: enabled
      cert-manager:
        status: enabled
      oauth2-proxy:
        status: enabled
```

### Configuration Options

| Field | Type | Description |
|---|---|---|
| `networking.type` | string | `ingress` (default) or `gateway`. Sets the active routing API for catalog templates. |
| `networking.ingress.className` | string | Ingress class name (defaults to `traefik`). Used when `type: ingress`. |
| `networking.gateway.name` | string | Name of the parent Gateway resource. Required when `type: gateway`. |
| `networking.gateway.namespace` | string | Namespace of the parent Gateway resource (must be `traefik` for the bundled Traefik setup). |
| `networking.gateway.sectionName` | string | Optional listener name on the Gateway (defaults to `websecure`). |

### Service-Level Overrides

Individual services can override the cluster Gateway reference or add custom routing annotations under `services.<service>.networking`:

```yaml
services:
  custom-service:
    status: enabled
    networking:
      gateway:
        name: custom-gateway
        namespace: custom-namespace
        sectionName: https
      annotations:
        custom.annotation/key: "value"
```

A service-level `gateway` override completely replaces the cluster Gateway reference for that service and requires `networking.type: gateway`.

---

## Workflow & Deployment

1. **Update `config.yaml`:**
   Set `networking.type: gateway` with the corresponding `gateway` block.

2. **Generate platform manifests:**
   Run `kubara generate` to re-render Helm values and manifests:
   ```bash
   kubara generate
   ```

3. **Review generated artifacts:**
   * In `platform-configs/<cluster>/helm/traefik/values.generated.yaml`, verify that `providers.kubernetesGateway.enabled: true` and the `gateway` block are rendered.
   * In platform services (`argo-cd`, `homer-dashboard`, `kube-prometheus-stack`, `longhorn`, `kyverno-policy-reporter`, `oauth2-proxy`), verify that `ingress.enabled: false` and `gatewayRoutes` / `httproute` are generated.

4. **Deploy via GitOps:**
   Commit and push your changes. Argo CD will synchronize:
   * Gateway API Standard CRDs (installed via `bootstrap-crds`).
   * Traefik Gateway, listeners, and certificates.
   * `HTTPRoute` resources attaching to the Gateway.
