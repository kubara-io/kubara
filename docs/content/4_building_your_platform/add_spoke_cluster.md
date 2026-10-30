# How to add a spoke cluster

After your hub cluster is running, you can add Kubernetes spoke clusters and manage them through Argo CD.

You can onboard:

* a new cluster provisioned with kubara Terraform templates, or
* an existing cluster you already operate.

The GitOps reconciliation and Argo CD integration flow are the same in both cases.

## Prerequisites

Before onboarding a spoke cluster, verify you have:

* A running hub cluster bootstrapped with Argo CD and `cluster-credential-rotator`.
* Administrative access to the hub cluster (`KUBECONFIG` pointing to the hub, for example `/tmp/hub.yaml`).
* An initial kubeconfig for the spoke cluster (for example `/tmp/spoke.yaml`) with sufficient permissions to create namespaces, ServiceAccounts, and cluster role bindings.
* Access to an External Secrets compatible secret manager or vault (such as HashiCorp Vault, OpenBao, STACKIT Secrets Manager, GCP Secret Manager, Azure Key Vault, AWS Secrets Manager, or Bitwarden). This can either be a separate instance for the spoke or the same instance used by the hub.

## 1. Add the spoke cluster to config.yaml

Run `cluster add` to register the new cluster:

```bash
kubara cluster add spoke-1
```

To assign custom or ordered catalogs, pass `--catalog /path/to/catalog` (repeat `--catalog` for multiple catalogs). Without `--catalog`, kubara assigns its default general catalog.

Spoke cluster entries are added with `type: spoke`. They do not contain an `argocd` section because GitOps reconciliation is centrally coordinated from the hub cluster.

### Optional: Ingress annotation overrides

Each service supports an optional `networking.annotations` map that is merged with kubara default annotations when rendering service values files.
This is useful when you use an ingress controller other than Traefik that requires controller-specific annotations.

```yaml
services:
  kube-prometheus-stack:
    status: enabled
    networking:
      annotations:
        nginx.ingress.kubernetes.io/auth-url: "https://$host/oauth2/auth"
        nginx.ingress.kubernetes.io/auth-signin: "https://$host/oauth2/start?rd=$escaped_request_uri"
```

User-provided annotations are merged on top of kubara defaults using `mergeOverwrite`. Values for the same key are overwritten, while kubara defaults not present in the override map are preserved.

## 2. Update config.yaml

Open `config.yaml` and make the necessary changes for your spoke cluster:

* Set the appropriate `stage` and cluster settings.
* Configure base domain and networking options.
* Enable or disable platform services (such as `cert-manager`, `external-secrets`, `traefik`, or `kube-prometheus-stack`).
* If you provision the spoke cluster through kubara Terraform, configure provider credentials and node pool specifications.

If this is a new cluster that you plan to provision using kubara Terraform, apply the Terraform configuration for the spoke cluster before continuing to onboarding. If the cluster already exists, continue directly to the next step.

## 3. Generate manifests

Generate the configuration artifacts:

```bash
kubara generate
```

This creates or updates:

* Spoke cluster overlays in `platform-configs/<spoke-cluster-name>/terraform/...` and `platform-configs/<spoke-cluster-name>/helm/...`
* Spoke cluster registration in the hub cluster's `cluster-credential-rotator` (an authoritative ConfigMap named `<spoke-cluster-name>-config` in `kubara-system`)
* Argo CD Application manifests on the hub that point to the spoke cluster

## 4. Commit and push changes

Commit and push all generated manifests to your Git repository:

```bash
git add .
git commit -m "feat: add spoke-1 cluster"
git push
```

Pushing these changes allows the hub cluster's Argo CD to reconcile the updated platform configuration. During reconciliation, Argo CD applies the `<spoke-cluster-name>-config` ConfigMap and prepares the `cluster-credential-rotator` CronJob on the hub.

If `argocd.selfManaged` is disabled on your hub cluster, run `kubara bootstrap <hub-cluster-name>` to apply the changes instead.

## 5. Onboard the spoke cluster

Once the hub reconciles the spoke configuration, onboard the spoke cluster by targeting the hub cluster with `KUBECONFIG` and passing the spoke's initial kubeconfig:

```bash
KUBECONFIG=/tmp/hub.yaml kubara bootstrap spoke-1 \
  --with-es-css-file css.yaml \
  --initial-kubeconfig /tmp/spoke.yaml
```

You can also pass `--kubeconfig /tmp/hub.yaml` instead of setting the environment variable. Re-running this command with a valid `--initial-kubeconfig` resets the spoke credential secret on the hub and restarts the onboarding flow if recovery is ever needed.

Output:

```text
2026-10-20 08:40:45 INF Adding helm repositories
2026-10-20 08:40:46 INF Refreshed helm repository index
2026-10-20 08:40:46 INF Updating helm chart dependencies
2026-10-20 08:40:47 INF Applying CRDs
2026-10-20 08:40:49 INF CRDs applied and established for: "bootstrap-crds"
2026-10-20 08:40:49 INF Applied ClusterSecretStore to spoke cluster
2026-10-20 08:40:49 INF Created initial cluster secret "spoke-1-cluster-secret"
2026-10-20 08:40:49 INF Waiting for credential rotator to onboard spoke "spoke-1"...
2026-10-20 08:41:05 INF Spoke cluster "spoke-1" onboarded successfully
```

### How onboarding works under the hood

``` mermaid
sequenceDiagram
    autonumber
    actor Operator as Operator (CLI)
    participant Hub as Hub (kubara-system)
    participant Rotator as Hub (Credential Rotator)
    participant Spoke as Spoke Cluster
    participant Argo as Hub (Argo CD)

    Operator->>Hub: kubara bootstrap spoke-1 (passes initial kubeconfig)
    Operator->>Spoke: Apply bootstrap CRDs (and optional ClusterSecretStore)
    Operator->>Hub: Upsert spoke-1-cluster-secret with bootstrapKubeconfig
    Rotator->>Hub: Scheduled CronJob picks up spoke-1-cluster-secret
    Rotator->>Spoke: Connect via initial kubeconfig
    Rotator->>Spoke: Create kubara-agent SA & ClusterRoleBinding in kubara-system
    Rotator->>Rotator: Mint short-lived rotating bearer token
    Rotator->>Hub: Wipe bootstrapKubeconfig, update spoke-1-cluster-secret & project into argocd
    Argo->>Spoke: Connect via rotating token & reconcile workloads
```

During the onboarding command:

1. **Spoke CRD and secret store setup**: `kubara` connects to the spoke cluster using `--initial-kubeconfig`, applies and waits for the required CRDs from the bootstrap catalog's `bootstrap-crds` service, and (if `--with-es-css-file` is provided) renders and applies the `ClusterSecretStore` manifest.
2. **Initial secret creation**: `kubara` connects to the hub cluster via `KUBECONFIG` and creates or resets the cluster credentials secret (`<spoke-name>-cluster-secret`) in `kubara-system`, storing `--initial-kubeconfig` under the `bootstrapKubeconfig` key.
3. **Scheduled rotator pickup**: `kubara` waits for the `cluster-credential-rotator` CronJob on the hub (running every 3 minutes) to process `<spoke-name>-cluster-secret`.
4. **Agent setup on the spoke**: The rotator uses the initial kubeconfig to connect to the spoke cluster, creates the `kubara-agent` ServiceAccount in `kubara-system`, and configures its cluster role binding.
5. **Credential minting, cleanup, and projection**: The rotator mints a 1-hour bearer token for `kubara-agent`, removes `bootstrapKubeconfig` from `<spoke-name>-cluster-secret`, marks `kubara.io/rotation-status: ready`, and projects the cluster secret into `argocd` so Argo CD can connect to the spoke.
6. **Continuous rotation**: The `cluster-credential-rotator` CronJob refreshes tokens expiring within 15 minutes. The initial kubeconfig is wiped from the hub after onboarding.

## 6. Configure External Secrets and roll out workloads

After Argo CD connects to the spoke cluster, it synchronizes platform services onto it. If enabled services require secrets (such as TLS certificates or ingress credentials), configure the External Secrets Operator on the spoke cluster to reach your secret manager.

Create provider credentials as a Kubernetes Secret on the spoke cluster, for example:

```bash
# Bitwarden
kubectl --kubeconfig /tmp/spoke.yaml -n external-secrets create secret generic bitwarden-access-token \
  --from-literal=token="<BITWARDEN_MACHINE_ACCOUNT_TOKEN>"

# STACKIT Secrets Manager
kubectl --kubeconfig /tmp/spoke.yaml -n external-secrets create secret generic stackit-secrets-manager-cred \
  --from-literal=username="<USERNAME>" \
  --from-literal=password="<PASSWORD>"
```

Then configure the spoke `ClusterSecretStore` in a chart overlay file, for example `platform-configs/<spoke-cluster-name>/helm/external-secrets/values-additional.yaml`:

```yaml
clusterSecretStores:
  workload-0-dev:
    labels:
      argocd.argoproj.io/instance: workload-0-external-secrets
    provider:
      vault:
        auth:
          userPass:
            path: userpass
            secretRef:
              name: stackit-secrets-manager-cred
              namespace: external-secrets
              key: password
            username: "<USERNAME>"
        path: "<VAULT_PATH>"
        server: "https://vault.example.com"
        version: v2
```

Alternatively, apply your `ClusterSecretStore` manifest (`css.yaml`) directly to the spoke cluster with `kubectl apply -f css.yaml --kubeconfig /tmp/spoke.yaml`.

Once the secret store connects to your secret backend, Argo CD reconciles all remaining workloads and dependencies on the spoke cluster.

![Hub_n_Spoke](../assets/diagrams.drawio)

### Fallback: Static ExternalSecrets

When `argocd.clusterSecrets.provider` is set to `external-secrets` on the hub cluster in `config.yaml` (or `clusterSecrets.useExternalSecrets: true` in Argo CD chart values), Argo CD renders static `ExternalSecret` manifests for spoke clusters instead of using `cluster-credential-rotator`.

In that mode, store the spoke kubeconfig in your secret backend (Vault/Secret Manager) under:
`<cluster-name>/<cluster-stage>/argocd/<spoke-name>-<spoke-stage>`
with the secret property named `kubeconfig`.

By default kubara keeps the original client-certificate based Argo CD cluster config for static ExternalSecrets.
If your kubeconfig should be used with bearer token auth in addition to the certificates, set
`bootstrapValues.cluster[].authMethod: token` in your Argo CD values. Note that `authMethod` must be exactly `token` or `clientCertificate` (defaults to `clientCertificate`); any other value triggers a template rendering error.

If Argo CD should only manage selected namespaces on the spoke cluster, set `bootstrapValues.cluster[].namespaces` to the allowed list of namespaces in your Argo CD values (as a YAML list):

```yaml
bootstrapValues:
  cluster:
    - name: spoke-dev
      namespaces:
        - team-a
        - team-b
```

## Additional notes

* If you enable `oauth2-proxy`, provide valid OAuth credentials in the secret backend used by external-secrets on the spoke cluster.
* Extra overlay files can use any `values-*.yaml` name. `values-additional.yaml` is a common choice that keeps provider-specific overrides separate from generated `values.generated.yaml` files.

### Alternative multi-cluster architectures

The standard spoke flow assumes the hub cluster can directly reach the spoke cluster API endpoint using stored credentials. For alternative topologies and fleet management patterns, see these reference setups in `platform-setup-examples`:

* [Argo CD Agent Managed-Mode Lab](https://github.com/kubara-io/platform-setup-examples/tree/main/argo-cd-agents): Outbound-only agent topology where spoke clusters initiate outbound mTLS connections to the hub and run their own application controllers, eliminating the need to store spoke kubeconfigs on the hub.
* [Scalable Cluster Management with Project Sveltos](https://github.com/kubara-io/platform-setup-examples/tree/main/sveltos-cluster-management): Offloads add-on and configuration lifecycle across large cluster fleets to Project Sveltos using declarative selectors.
