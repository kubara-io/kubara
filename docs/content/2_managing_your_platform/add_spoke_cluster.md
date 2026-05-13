# Add Spoke Cluster

After your hub cluster is running, you can add more Kubernetes spoke clusters and manage them through Argo CD.

You can onboard:

* a new cluster provisioned with kubara Terraform templates, or
* an existing cluster you already operate.

The Argo CD integration flow is the same in both cases.

## 1. Add spoke cluster to `config.yaml`

Add a new cluster entry:

```yaml
clusters:
  - name: workload-0
    stage: dev
    type: spoke
    dnsName: workload-0.dev.example.com
    ssoOrg: my-org
    ssoTeam: my-team
    terraform:
      provider: stackit # currently supported: stackit
      projectId: <project-id>
      kubernetesType: ske
      kubernetesVersion: 1.34
      dns:
        name: workload-0.dev.example.com
        email: platform@example.com
    argocd:
      repo:
        https:
          customer:
            url: https://git.example.com/platform/repo.git
            targetRevision: main
          managed:
            url: https://git.example.com/platform/repo.git
            targetRevision: main
    services:
      argo-cd:
        status: enabled
      cert-manager:
        status: enabled
        config:
          clusterIssuer:
            name: letsencrypt-staging
            email: platform@example.com
            server: https://acme-staging-v02.api.letsencrypt.org/directory
      external-dns:
        status: enabled
      external-secrets:
        status: enabled
      kube-prometheus-stack:
        status: enabled
        config:
            storageClassName: standard-rwo # optional
      traefik:
        status: enabled
      kyverno:
        status: enabled
      kyverno-policies:
        status: enabled
      kyverno-policy-reporter:
        status: enabled
      loki:
        status: enabled
        config:
            storageClassName: standard-rwo # optional
      homer-dashboard:
        status: enabled
      oauth2-proxy:
        status: enabled
      metrics-server:
        status: disabled
      metallb:
        status: disabled
      longhorn:
        status: disabled
```

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

## 3. Prepare the spoke cluster

If this is a new cluster, apply Terraform for the spoke entry.
If the cluster already exists, skip Terraform and continue.

You need the spoke cluster kubeconfig for registration in Argo CD.
Store it in your secret backend (Vault/Secret Manager), for example:

```json
{
  "my_clusters": {
    "k8s-spoke-0": "<spoke kubeconfig yaml>"
  }
}
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

## 5. Register spoke cluster in Argo CD

Update the hub cluster overlay:
`customer-service-catalog/helm/<hub-cluster-name>/argo-cd/values.yaml`

```yaml
bootstrapValues:
  cluster:
    - name: my-new-spoke-0
      project: hub-production
      remoteRef:
        remoteKey: my_clusters
        remoteKeyProperty: k8s-spoke-0
      secretStoreRef:
        kind: ClusterSecretStore
        name: <hub-cluster-name>-<stage>
      additionalLabels:
        cert-manager: enabled
        external-dns: enabled
        external-secrets: enabled
        traefik: enabled
        kube-prometheus-stack: enabled
        kyverno: enabled
        kyverno-policies: enabled
        kyverno-policy-reporter: enabled
        loki: enabled
        oauth2-proxy: enabled
```

The `remoteRef` points to the spoke kubeconfig secret in your secret backend.

![Add Workload Cluster](../images/add-workload-cluster.png)

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
