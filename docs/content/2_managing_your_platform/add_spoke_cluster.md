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

This will create a new cluster in the config.yaml that will respect your catalog choices.

### Optional: Ingress annotation overrides

Each service supports an optional `ingress.annotations` map that is merged with kubara's default annotations when rendering the service values files.
This is useful when you use an ingress controller other than Traefik that requires controller-specific annotations.

```yaml
services:
  kubePrometheusStack:
    status: enabled
    ingress:
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

* `customer-service-catalog/terraform/<spoke-cluster-name>/...`
* `customer-service-catalog/helm/<spoke-cluster-name>/...`

This also registers the spokes in the ArgoCD `values.yaml`.

## 3. Prepare the spoke cluster

If this is a new cluster, apply Terraform for the spoke entry.
If the cluster already exists, skip Terraform and continue.

You need the spoke cluster kubeconfig for registration in Argo CD.
Store it in your secret backend (Vault/Secret Manager), for example:
The config expects the kubeconfig to be reachable under:
<cluster-name>/<cluster-stage>/argocd/<spoke-name>-<spoke-stage>
And the secret itself simply be named `kubeconfig`.


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

Then configure the spoke ClusterSecretStore in:
`customer-service-catalog/helm/<spoke-cluster-name>/external-secrets/additional-values.yaml`

Example:

```yaml
clusterSecretStores:
  - name: workload-0-dev
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

## 6. Commit and roll out

Commit and push all updated files.

If Argo CD manages itself, it will reconcile automatically.
If not, run bootstrap again for your hub cluster:

```bash
kubara bootstrap <hub-cluster-name-from-config-yaml>
```

## Additional notes

* If you enable `oauth2-proxy`, provide valid OAuth credentials in the secret backend used by external-secrets on the spoke cluster.
* `additional-values.yaml` is optional but recommended for provider-specific overrides, because generated `values.yaml` can be re-rendered by kubara.
