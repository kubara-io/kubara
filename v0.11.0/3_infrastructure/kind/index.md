# kind Local Evaluation

This page explains in detail how the local `kind` setup behind `kubara bootstrap --local` works. This is only meant for the curious minded explores. You don't need to understand these details to use kubara or the local test setup.

If you only want the fast hands-on path, use the [Quick Start](../1_getting_started/quick_start.md). This page is about the mechanics and the local-only changes kubara applies for you.

## What this preset is for

The `kind` preset is an **evaluation environment**, not a production pattern.

It exists so you can try kubara quickly on one local machine without first provisioning:

- a managed Kubernetes cluster
- a DNS provider integration
- a production-grade secret manager

Instead, kubara builds a small local setup around:

- `kind` for the Kubernetes cluster
- `cloud-provider-kind` for `LoadBalancer` IP assignment
- Traefik for ingress
- OpenBao in dev mode as a temporary secret backend for `external-secrets`

## Host prerequisites

`kubara bootstrap --local` checks for these binaries on your host:

- `kind`
- `docker`
- `kubectl`
- `helm`
- `cloud-provider-kind`

The Quick Start also assumes you can run `sudo cloud-provider-kind`, because kubara waits for you to start it manually once the placeholder `LoadBalancer` service exists.

## What kubara writes locally

The local runtime data goes into `.local/` in your GitOps repository:

- `.local/kind.kubeconfig`: kubeconfig exported from `kind`
- `.local/kind-config.yaml`: the generated `kind` cluster definition
- `.local/generate.env`: environment snapshot used for the local `kubara generate --helm` rerun
- `.local/openbao/values.yaml`: Helm values for the OpenBao dev setup
- `.local/external-secrets/clustersecretstore.yaml`: local `ClusterSecretStore` manifest

kubara also ensures `.gitignore` contains the local runtime entries so these files do not become part of your intended GitOps state.

## Bootstrap sequence

### 1. Create or reuse the kind cluster

kubara writes a `kind` config file and then creates the cluster if it does not exist yet.
If the cluster already exists, kubara reuses it and exports the kubeconfig again.

The generated `kind` config mounts the host CA bundle into the control plane node at `/etc/ssl/certs/ca-certificates.crt`.
That is done so workloads inside the cluster can trust the same CA bundle when they need outbound TLS access.

### 2. Create a placeholder Traefik LoadBalancer service

Before the normal Helm deployment finishes, kubara creates:

- The `traefik` namespace
- A placeholder `Service` called `traefik`
- Service ports `80` and `443`
- Service type `LoadBalancer`

kubara then waits until the service gets an external IP.

At this point it prints the instruction to start:

```bash
sudo cloud-provider-kind
```

`cloud-provider-kind` assigns a local IP from the Docker network to the Traefik `LoadBalancer` service.

### 3. Derive the local DNS names

Once the Traefik service has a `LoadBalancer` IP, kubara derives:

- `<load-balancer-ip>.traefik.me` as the base host
- `openbao.<load-balancer-ip>.traefik.me` for OpenBao

That is why the local evaluation URLs look like:

- `https://<ip>.traefik.me/argocd`
- `https://openbao.<ip>.traefik.me/ui`

### 4. Rewrite the cluster profile for local mode

kubara loads your `config.yaml`, finds the selected cluster, and applies the local profile before regenerating Helm output.

The local profile does all of the following:

- Forces `type: hub`
- Sets `dnsName` to the generated `traefik.me` host
- Sets `ssoOrg` and `ssoTeam` to `local`
- Forces `ingressClassName: traefik`
- Only enables a minimal set of services:
  - `argocd`
  - `cert-manager`
  - `external-secrets`
  - `homer-dashboard`
  - `kube-prometheus-stack`
  - `metrics-server`
  - `traefik`

After changing the cluster profile, kubara writes `.local/generate.env` and reruns `kubara generate --helm` so the rendered Helm artifacts match the local profile.

### 5. Install and configure OpenBao

kubara installs the OpenBao Helm chart with local values that:

- Enable OpenBao dev mode
- Use the fixed dev root token `root`
- Expose OpenBao through Traefik without TLS
- Disable the injector
- Enable the UI

After the pod is ready, kubara configures OpenBao for the local `external-secrets` flow:

1. Enable the `kv-v2` secret engine at `kv/`
2. Enable the Kubernetes auth method
3. Configure Kubernetes auth against the in-cluster API server
4. Create a read-only policy for `kv/data/*` and `kv/metadata/*`
5. Create a very permissive Kubernetes role named `any-sa`

That role is intentionally wide open for local evaluation. It is not a production security model.

kubara then writes at least these secrets into OpenBao:

- Grafana admin credentials at `<cluster>/<stage>/kube-prometheus-stack/grafana_credentials`
- Optionally the Docker pull secret at `<cluster>/<stage>/cluster_secrets/docker_config` when `DOCKERCONFIG_BASE64` is present

### 6. Generate local overlay files

kubara writes extra local overlay values so the generated platform is easier to run on a small local cluster:

- Argo CD:
    - Disables Dex
    - Enables simpler ingress handling
    - Mounts the host CA bundle into the repo server
    - Sets permissive default RBAC for local evaluation
    - Reduces resource usage
- cert-manager:
    - Disables Let's Encrypt
    - Creates a self-signed root issuer instead
- Platform UIs:
    - Enables simple ingress overrides for Homer, Prometheus, Alertmanager, Grafana, Kyverno UI, Longhorn, and the Traefik dashboard
- OAuth2 Proxy:
    - Removes the local override file entirely so this local flow stays simpler

kubara also writes a local `ClusterSecretStore` manifest that points `external-secrets` at:

- Server: `http://openbao.openbao.svc:8200`
- Path: `kv`
- Auth mount: `kubernetes`
- Role: `any-sa`

### 7. Continue with the normal bootstrap

After the local preparation is done, the remaining bootstrap flow continues with the same general kubara bootstrap logic:

- Install required CRDs when requested
- Install Argo CD
- Hand ongoing reconciliation over to Argo CD

## Important local-only tradeoffs

This setup is intentionally relaxed:

- OpenBao runs in dev mode
- The root token is always `root`
- The secret data is in-memory only
- The Kubernetes auth role is intentionally broad
- Local ingress uses `traefik.me` convenience hostnames

If OpenBao loses its pod or runtime state, you usually need to rerun:

```bash
kubara bootstrap --local test-cluster
```

## When to use a different path

Use one of the other infrastructure presets, or your own infrastructure, when you need:

- Durable secrets management
- Provider-managed DNS
- A real cloud or edge ingress design
- Multi-node production-grade operations

For the guided hands-on flow, go back to the [Quick Start](../1_getting_started/quick_start.md).
