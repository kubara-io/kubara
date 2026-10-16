# How to add a Spoke Cluster

After your hub cluster is running, you can add more Kubernetes spoke clusters and manage them through Argo CD.

You can onboard:

* a new cluster provisioned with kubara Terraform templates, or
* an existing cluster you already operate.

The Argo CD integration flow is the same in both cases.

## 1. Add spoke cluster to `config.yaml`

To add a new cluster to your `config.yaml` simply type:
```bash
kubara cluster add <spoke-name> --catalog /path/to/catalog
```

Repeat `--catalog` to assign multiple ordered catalogs. The command stores those references in the new cluster's `catalogs` list. Without `--catalog`, it assigns kubara's general catalog. Spoke cluster entries do not contain an `argocd` section, as GitOps reconciliation is centrally coordinated from the hub cluster.

Later `schema`, `generate`, and `bootstrap` runs automatically resolve the catalogs configured for that spoke. Additional `--catalog` values are appended temporarily and are not persisted.

### Optional: Ingress annotation overrides

Each service supports an optional `networking.annotations` map that is merged with kubara's default annotations when rendering the service values files.
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

User-provided annotations are merged on top of kubara defaults using `mergeOverwrite`: values for the same key are overwritten, but kubara defaults that are not present in the override map are preserved.

## 2. Regenerate Terraform and Helm templates

```bash
# kubara generate creates both helm and terraform files by default
kubara generate
```

This creates/updates the spoke cluster overlays in:

* `platform-configs/<spoke-cluster-name>/terraform/...`
* `platform-configs/<spoke-cluster-name>/helm/...`

This also registers the spoke cluster in the hub's `cluster-credential-rotator` and `argo-cd` generated values.

## 3. Commit, roll out hub changes, and onboard the spoke cluster

If this is a new cluster, apply Terraform for the spoke entry first. If the cluster already exists, skip Terraform and continue.

Commit and push the generated files so that the hub reconciles the spoke configuration (or run `kubara bootstrap <hub-cluster-name>` if `selfManaged` is disabled).

Once the hub has reconciled `cluster-credential-rotator` and the `<spoke-name>-config` ConfigMap in `kubara-system`, onboard the spoke cluster using its initial kubeconfig:

```bash
kubara bootstrap <spoke-name> \
  --kubeconfig /path/to/hub.kubeconfig \
  --initial-kubeconfig /path/to/spoke.kubeconfig
```

You can optionally customize the namespace for the `kubara-agent` ServiceAccount on the spoke cluster (defaults to `kubara`):

```bash
kubara bootstrap <spoke-name> \
  --kubeconfig /path/to/hub.kubeconfig \
  --initial-kubeconfig /path/to/spoke.kubeconfig \
  --namespace kubara
```

During onboarding, `cluster-credential-rotator` on the hub uses the initial kubeconfig once to provision the `kubara-agent` ServiceAccount on the spoke, mints short-lived rotating credentials in `kubara-system`, and projects the target cluster secret into `argocd`.

### Fallback: Static ExternalSecrets (when `selfManaged: disabled`)

When `argocd.selfManaged` is set to `disabled` on the hub cluster (or `clusterSecrets.useExternalSecrets: true`), Argo CD renders static `ExternalSecret` manifests for spoke clusters instead of using `cluster-credential-rotator`.

In that mode, store the spoke kubeconfig in your secret backend (Vault/Secret Manager) under:
`<cluster-name>/<cluster-stage>/argocd/<spoke-name>-<spoke-stage>`
with the secret property named `kubeconfig`.

By default kubara keeps the original client-certificate based Argo CD cluster config for static ExternalSecrets.
If your kubeconfig should be used with bearer token auth in addition to the certificates, set
`bootstrapValues.cluster[].authMethod: token` in your Argo CD values. Note that `authMethod` must be exactly `token` or `clientCertificate` (defaults to `clientCertificate`); any other value will trigger a template rendering error.

If Argo CD should only manage selected namespaces on the spoke cluster, set `bootstrapValues.cluster[].namespaces` to the allowed list of namespaces in your Argo CD values (as a YAML list):

```yaml
bootstrapValues:
  cluster:
    - name: spoke-dev
      namespaces:
        - team-a
        - team-b
```

## 4. Prepare external-secrets credentials on the spoke cluster

Create provider credentials as Kubernetes secret(s), for example:

```bash
# Bitwarden
kubectl -n external-secrets create secret generic bitwarden-access-token \
  --from-literal=token="<BITWARDEN_MACHINE_ACCOUNT_TOKEN>"

# STACKIT Secrets Manager
kubectl -n external-secrets create secret generic stackit-secrets-manager-cred \
  --from-literal=username="<USERNAME>" \
  --from-literal=password="<PASSWORD>"
```

Then configure the spoke ClusterSecretStore in a chart overlay file, for example:
`platform-configs/<spoke-cluster-name>/helm/external-secrets/values-additional.yaml`

Example:

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

![Hub_n_Spoke](../assets/diagrams.drawio)

## 5. Roll out spoke workloads

Once the spoke is onboarded and external-secrets is configured, Argo CD will automatically reconcile all enabled platform services onto the spoke cluster.

## Additional notes

* If you enable `oauth2-proxy`, provide valid OAuth credentials in the secret backend used by external-secrets on the spoke cluster.
* Extra overlay files can use any `values-*.yaml` name. `values-additional.yaml` is a common choice, and it keeps provider-specific overrides separate from the generated `values.generated.yaml`.

### Alternative multi-cluster architectures

The standard spoke flow assumes the hub cluster can directly reach the spoke cluster's Kubernetes API endpoint using stored credentials. For alternative topologies and fleet management patterns, see the following reference setups in `platform-setup-examples`:

* **Outbound-only agent topology**: [Argo CD Agent Managed-Mode Lab](https://github.com/kubara-io/platform-setup-examples/tree/main/argo-cd-agents) demonstrates a hub-and-spoke setup where spoke clusters initiate outbound mTLS connections to the hub and run their own application controllers, eliminating the need to store spoke kubeconfigs on the hub.
* **Fleet add-on distribution**: [Scalable Cluster Management with Project Sveltos](https://github.com/kubara-io/platform-setup-examples/tree/main/sveltos-cluster-management) demonstrates delegating add-on and configuration lifecycle across large cluster fleets to Project Sveltos using declarative selectors.
