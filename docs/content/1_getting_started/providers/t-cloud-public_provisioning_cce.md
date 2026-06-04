# T Cloud Public CCE Provisioning

For T Cloud Public, kubara generates a CCE-focused Terraform setup composed from small modules.

## Generated Infrastructure

Running `kubara generate --terraform` creates:

- `objectstorage-bucket`: OBS bucket plus dedicated S3-compatible credentials for Terraform state or backup buckets
- `identity-agencies`: IAM agencies for tenant-level service authorizations
- `dns-zone`: DNS zone for the configured cluster domain
- `network`: VPC, subnet, optional NAT gateway, shared external load balancer and optional independent dedicated load balancer
- `keypair`: SSH keypair for CCE node pools
- `kms-key`: KMS key for encrypted node volumes
- `cce-cluster`: CCE cluster, configurable node pools, optional CCE addons, and optional local kubeconfig output
- `openbao-helm`: OpenBao Helm release deployed after the CCE cluster is available
- `openbao`: separate Terraform layer for OpenBao KV, External Secrets access, and platform secrets

OpenBao is the in-cluster secret backend for the T Cloud Public setup. kubara deploys it because this provider path does not currently integrate a managed Vault-compatible secret backend. OpenBao stores platform secrets and provides them to workloads through External Secrets.

The same bucket module is also used for Velero backup buckets. The generated customer infrastructure renders a `velero_bucket` module only when Velero is enabled and `services.velero.config.backupStorage.create` is `true`.

Generated OBS buckets use KMS server-side encryption by default. The bootstrap state backend stack normally creates the required tenant-wide OBS KMS agency. If you skipped that stack, enable `create_obs_kms_agency` before creating encrypted buckets. Set `create_t_cloud_public_agencies = false` if all required IAM agencies already exist.

## 1. Review Generated Values

Review `customer-service-catalog/terraform/<cluster-name>/infrastructure/env.auto.tfvars`. Keep persistent changes in a separate override file as described in [Terraform value overrides](../overview_core_concept.md#terraform-value-overrides).

A few defaults that often need attention before the first apply:

- **`enable_cluster_public_endpoint = true`** binds a small EIP (`5_bgp`, 5 Mbit/s, traffic-charged) to the CCE master so the API server is reachable from the machine that runs Terraform — required for the in-stack Helm provider (e.g. the OpenBao Helm release) when applying from outside the VPC. After apply, the public IP is exposed as the `cluster_public_endpoint_ip` output. Set to `false` if you only run Terraform from inside the VPC (CI runner inside OTC, bastion, VPN).

- **`enable_nat_gateway = true`** creates the NAT gateway for node egress. Adjust `nat_gateway_spec` and `nat_eip_bandwidth_size` if the default size does not fit your cluster.
- **`enable_shared_load_balancer = true`** creates the shared ELB used by Traefik. Set `enable_dedicated_load_balancer = true` only if you also need an independent dedicated ELB.
- **`enable_openbao = true`** rolls out the in-cluster OpenBao Helm release after CCE comes up. The release is **not** initialized or unsealed automatically — that happens manually in step 4 below.

## 2. Initialize Backend

Change into the generated Terraform directory and source its environment:

```bash
cd customer-service-catalog/terraform/<cluster-name>
source set-env.sh
cd infrastructure
```

The generated infrastructure backend stores state in:

```hcl
bucket = "bucket-tf-<cluster-name>-<stage>"
key    = "tf-state-<cluster-name>-<stage>"
```

The default backend endpoint is `https://obs.eu-de.otc.t-systems.com`, which is the technical OBS endpoint for the `eu-de` region. For `eu-nl`, adjust the generated backend endpoint and region before running `terraform init`.

Make sure `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` from the bootstrap stack are set in `set-env.sh` before initializing the backend. The script also sets the AWS SDK checksum compatibility options required for the T Cloud Public OBS S3-compatible backend.

## 3. Apply Infrastructure

Run:

=== "Terraform"

    ```bash
    terraform init
    terraform plan
    terraform apply
    ```

=== "Tofu"

    ```bash
    tofu init
    tofu plan
    tofu apply
    ```

The generated stack can optionally write a kubeconfig locally when `create_kubeconfig_local` is enabled.

## 4. OpenBao Manual Init and Unseal

After the infrastructure apply has installed the OpenBao Helm release, OpenBao is running as a 3-replica HA Raft cluster but sealed. Until it is initialized and unsealed, the pods report `0/1` ready, which is expected: the readiness probe deliberately fails on a sealed pod.

The generated Helm release configures Raft peer discovery (`retry_join`) so the replicas find each other automatically. You only need to run `init` once on the first pod, then `unseal` on each replica.

### 1. Wait for the pods to be Running

```bash
kubectl -n openbao get pods -w
```

You should see `openbao-0`, `openbao-1`, `openbao-2` reach `Running` state (still `0/1`). Press Ctrl-C once they all show `Running`.

### 2. Initialize OpenBao on the first pod

This generates the unseal keys and the initial root token. **Run this exactly once.** The output appears only once and cannot be recovered later.

```bash
kubectl exec -n openbao -ti openbao-0 -- bao operator init
```

Save the output immediately in an approved secure system that is accessible to the responsible team. Do not store unseal keys or the root token in Git, Terraform state, or shared plaintext files. By default OpenBao prints **5 unseal keys** and **1 initial root token**:

- Any **3 of the 5** unseal keys are needed to unseal a pod (Shamir threshold 3-of-5).
- Keep the keys under separate trusted operators or access controls where possible — anyone who collects 3 keys can decrypt the cluster.
- The root token gives full access; rotate or revoke it after creating a long-lived admin role.

### 3. Unseal each pod with 3 keys

Unsealing is per-pod and per-restart. Repeat the command **3 times per pod**, each time with a different unseal key:

```bash
# Pod 0
kubectl exec -n openbao -ti openbao-0 -- bao operator unseal <unseal-key-1>
kubectl exec -n openbao -ti openbao-0 -- bao operator unseal <unseal-key-2>
kubectl exec -n openbao -ti openbao-0 -- bao operator unseal <unseal-key-3>

# Pod 1 — retry_join already attached it to the cluster as a follower
kubectl exec -n openbao -ti openbao-1 -- bao operator unseal <unseal-key-1>
kubectl exec -n openbao -ti openbao-1 -- bao operator unseal <unseal-key-2>
kubectl exec -n openbao -ti openbao-1 -- bao operator unseal <unseal-key-3>

# Pod 2
kubectl exec -n openbao -ti openbao-2 -- bao operator unseal <unseal-key-1>
kubectl exec -n openbao -ti openbao-2 -- bao operator unseal <unseal-key-2>
kubectl exec -n openbao -ti openbao-2 -- bao operator unseal <unseal-key-3>
```

After the third successful key on a pod, that pod becomes unsealed and ready.

### 4. Verify the cluster

```bash
kubectl exec -n openbao -ti openbao-0 -- bao status
```

Expected output:

```text
Sealed         false
Initialized    true
HA Enabled     true
HA Cluster     http://openbao-0.openbao-internal:8201
Active Node    true
```

`kubectl -n openbao get pods` should now show all three pods as `1/1` ready.

### 5. Make the root token available for the OpenBao Terraform layer

The Initial Root Token from step 2 is what the next stack (`openbao/`) uses to authenticate against OpenBao through the local port-forward. There is intentionally **no Terraform output** for it, so it is not stored in Terraform state.

The set-env script already contains commented-out lines for it. Uncomment them and fill in the token you saved:

```bash
# customer-service-catalog/terraform/<cluster-name>/set-env.sh
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="hvb.AAAAAQ..."   # Initial Root Token from `bao operator init`
```

Re-source the file so the new variables apply to the OpenBao Terraform layer:

```bash
source ../set-env.sh
```

### Pod restarts and auto-unseal

Until OpenBao supports auto-unseal with T Cloud Public KMS, every restarted OpenBao pod must be unsealed again with 3 of the 5 keys. This includes restarts caused by cluster upgrades, node maintenance, and OpenBao updates. The [T Cloud Public KMS wrapper](https://github.com/openbao/go-kms-wrapping/pull/63) has been merged, while the [OpenBao auto-unseal integration](https://github.com/openbao/openbao/issues/2302) is still pending.

Do not write OpenBao secrets in this first infrastructure apply. Apply OpenBao configuration and secrets in a separate step after OpenBao is initialized, unsealed, and a valid OpenBao token is available.

## 5. OpenBao Configuration and Secrets

kubara also renders a separate OpenBao Terraform layer:

```text
customer-service-catalog/terraform/<cluster-name>/openbao
```

This layer uses the same OBS backend bucket as the infrastructure layer, but stores state under a separate key:

```hcl
key = "tf-state-<cluster-name>-<stage>-openbao"
```

Run the port-forward in a separate terminal after OpenBao is initialized and unsealed:

```bash
kubectl -n openbao port-forward svc/openbao 8200:8200
```

Then apply the OpenBao Terraform layer:

```bash
cd ../openbao
terraform init
terraform apply
```

The layer configures a KV v2 mount, Kubernetes auth at `k8s-auth`, the namespace-scoped `k8s-kv-read` role and templated policy, the `external-secrets` role (used only for the cluster-wide image pull secret and limited to `secret/docker_config`), and the generated Grafana admin credentials.

User-provided secrets — the OAuth2 client credentials, `t-cloud-public-clouds-yaml` for ExternalDNS, and the Velero S3 credentials — are written through a separate `secrets.tf-example` file. Copy it to activate the blocks you need:

```bash
cp secrets.tf-example secrets.tf
```

Each block declares a `variable` and the matching `vault_kv_secret_v2` resource; the values come from `TF_VAR_*` environment variables in your sourced `set-env.sh` (which has commented-out templates for each one), so no secret is ever written into a committed file. Delete the blocks you do not use before applying.

### Namespace-isolated secret access

T Cloud Public uses **namespace-isolated** secret access. Each consuming service reads only its own namespace's secrets:

| Secret | KV path | Consuming namespace |
|--------|---------|---------------------|
| Grafana admin / OAuth2 | `secret/kube-prometheus-stack/*` | `kube-prometheus-stack` |
| Argo CD OAuth2 | `secret/argocd/*` | `argocd` |
| OAuth2 Proxy | `secret/oauth2-proxy/*` | `oauth2-proxy` |
| Velero S3 | `secret/velero/*` | `velero` |
| ExternalDNS clouds.yaml | `secret/external-dns/*` | `external-dns` |
| **Image pull secret** | `secret/docker_config` (flat) | **all** (cluster-wide) |

Every chart renders a `SecretStore` in its own namespace that authenticates with that namespace's `default` ServiceAccount through the `k8s-kv-read` role. The templated policy resolves the token to `secret/<namespace>/*`, so a workload in one namespace cannot read another namespace's secrets. The image pull secret is the deliberate exception: it is distributed to every namespace through a `ClusterExternalSecret`, so it stays on the cluster-wide store at a flat path.

Velero reads its S3 credentials through an `ExternalSecret` in the `velero` namespace. Keep the separate `external-secrets` service enabled; the Velero chart consumes External Secrets CRDs but does not install the External Secrets Operator itself. The generated `BackupStorageLocation` points at the same synchronized Kubernetes Secret (`velero-credentials`, key `cloud`).
With `services.velero.config.backupStorage.create: true`, the generated Velero values point at the Terraform-managed bucket name `velero-<cluster-name>-<stage>`. Set the matching `backupStorage.region` and `backupStorage.s3Url` values in the cluster config. If you use an existing OBS or S3-compatible bucket instead, set `backupStorage.create: false` and provide `backupStorage.bucketName`.
When Velero uses CSI snapshots (`backupMode: csi-snapshot` or `backupMode: csi-data-mover`), the generated values select the `t-cloud-public` `VolumeSnapshotClass` mapping for the CCE Everest CSI disk driver.

### OIDC admin access

The OpenBao Terraform layer can configure OIDC admin login. Put the overrides in `customer-service-catalog/terraform/<cluster-name>/openbao/override.auto.tfvars`, not in the generated `env.auto.tfvars`; see [Terraform value overrides](../overview_core_concept.md#terraform-value-overrides). For example, use the following values for a Keycloak client:

```hcl
manage_openbao_oidc_auth_backend = true
openbao_oidc_discovery_url       = "https://<keycloak-host>/realms/<realm>"
openbao_oidc_client_id           = "<openbao-client-id>"
openbao_oidc_admin_allowed_redirect_uris = [
  "https://<cluster-dns-name>/openbao/ui/vault/auth/oidc/oidc/callback",
  "https://<cluster-dns-name>/ui/vault/auth/oidc/oidc/callback",
  "http://127.0.0.1:8200/ui/vault/auth/oidc/oidc/callback",
]
```

Set `TF_VAR_openbao_oidc_client_secret` in `set-env.sh`, source the file again, and apply the OpenBao Terraform layer. After verifying OIDC access, revoke the Initial Root Token with `bao token revoke -self` and remove `VAULT_TOKEN` from `set-env.sh`.

## 6. Continue With Platform Bootstrap

From the repository root, export the kubeconfig:

```bash
cd customer-service-catalog/terraform/<cluster-name>/infrastructure
terraform output -raw kubeconfig_raw > ~/.kube/<cluster-name>.yaml
```

Continue with [Bootstrap Your Own Platform](../bootstrapping.md).
