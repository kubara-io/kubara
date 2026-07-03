# T Cloud Public (Community)

The terraform modules for the T Cloud Public are built by the kubara community and aren't tested on a regular basis through integration nor regression tests by the kubara maintainers.

The kubara provider key is `t-cloud-public` and the Kubernetes type is `cce` for Cloud Container Engine.

## Configuration

Use these values in `config.yaml`:

```yaml
terraform:
  provider: t-cloud-public
  projectId: <tenant-name>
  kubernetesType: cce
  kubernetesVersion: 1.29
  dns:
    name: <dns-name>
    email: <email>
```

For T Cloud Public, set `projectId` to the tenant/project name used as `tenant_name`, not to a UUID.

## 1. Generate Terraform modules

```bash
kubara generate --terraform
```

Commit and push the generated files to your Git repository.

The generated T Cloud Public Terraform layout contains:

- `bootstrap-tfstate-backend`: OBS bucket, backend credentials, and optional OBS KMS agency setup for Terraform state
- `infrastructure`: DNS zone, IAM agencies, VPC/subnet/NAT/load balancer, keypair, KMS key, CCE cluster, StorageClass resources, and the OpenBao Helm release
- `openbao`: OpenBao KV, Kubernetes auth, namespace-scoped policies and roles, and templated platform secrets
- reusable managed modules for OBS buckets, IAM agencies, DNS, network, keypair, KMS, CCE, and StorageClasses

The generated infrastructure stack is composed from small modules:

- `objectstorage-bucket`: OBS bucket plus dedicated S3-compatible credentials for Terraform state or backup buckets
- `identity-agencies`: IAM agencies for tenant-level service authorizations
- `dns-zone`: DNS zone for the configured cluster domain
- `network`: VPC, subnet, optional NAT gateway, shared external load balancer and optional independent dedicated load balancer
- `keypair`: SSH keypair for CCE node pools
- `kms-key`: KMS key for encrypted node volumes
- `cce-cluster`: CCE cluster, configurable node pools, optional CCE addons, and optional local kubeconfig output
- `openbao-helm`: OpenBao Helm release deployed after the CCE cluster is available

OpenBao is the in-cluster secret backend for the T Cloud Public setup. kubara deploys it because this provider path does not currently integrate a managed Vault-compatible secret backend. OpenBao stores platform secrets and provides them to workloads through External Secrets.

The same bucket module is also used for Velero backup buckets. The generated customer infrastructure renders a `velero_bucket` module only when Velero is enabled and `services.velero.config.backupStorage.create` is `true`.

Generated OBS buckets use KMS server-side encryption by default. The bootstrap state backend stack normally creates the required tenant-wide OBS KMS agency. If you skipped that stack, enable `create_obs_kms_agency` before creating encrypted buckets. Set `create_t_cloud_public_agencies = false` if all required IAM agencies already exist.

## 2. Prepare environment variables

Before the first `terraform init`, prepare and load your environment variables:

```bash
cd platform-configs/<cluster-name>/terraform
cp set-env-changeme.sh set-env.sh
```

Set the T Cloud Public provider variables in `set-env.sh` / `set-env.ps1` before sourcing:

```bash
export TF_VAR_t_cloud_public_region="eu-de"
export TF_VAR_t_cloud_public_domain_name="<domain-name>"
export TF_VAR_t_cloud_public_tenant_name="<tenant-name>"
export TF_VAR_t_cloud_public_access_key="<access-key>"
export TF_VAR_t_cloud_public_secret_key="<secret-key>"
```

`TF_VAR_t_cloud_public_tenant_name` must be the T Cloud Public tenant/project name. This is the same value kubara reads from `terraform.projectId` in `config.yaml` for T Cloud Public, and it is not a UUID.

The generated environment file also sets `AWS_REQUEST_CHECKSUM_CALCULATION=when_required` and `AWS_RESPONSE_CHECKSUM_VALIDATION=when_required`. Keep these values for the T Cloud Public OBS backend. Terraform's `s3` backend only supports S3-compatible APIs on a best-effort basis, and recent AWS SDK checksum behavior can otherwise produce unreadable state objects against some S3-compatible backends.

Then load the file:

```bash
source set-env.sh
# or for PowerShell
# cp set-env-changeme.ps1 set-env.ps1
# . .\set-env.ps1
```

## 3. Create the Terraform backend state

Then navigate to:

```bash
cd bootstrap-tfstate-backend
```

For persistent changes to generated Terraform values, use a separate override file as described in [Terraform value overrides](../2_concepts/overview_core_concept.md#terraform-value-overrides).

### OBS-to-KMS agency

The bootstrap stack creates an IAM agency named `OBSAccessKMS` (delegated to the `op_svc_obs` service principal) so that OBS can use the generated KMS key for server-side bucket encryption. Without this agency, the bucket creation fails with `Status=403 Forbidden, Code=AccessDenied` the moment `server_side_encryption` is set.

The agency is tenant-scoped and only needs to exist once per T Cloud Public tenant. If your tenant already has it, for example created out-of-band or from a previous bootstrap of another cluster, disable the in-stack creation:

```hcl
create_obs_kms_agency = false
```

If you do not need server-side encryption at all for the state bucket, set:

```hcl
enable_bucket_server_side_encryption = false
```

The agency module is then skipped entirely.

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

Use the output to configure Terraform backend credentials:

=== "Terraform"

    ```bash
    export AWS_ACCESS_KEY_ID="$(terraform output -raw credential_access_key)"
    export AWS_SECRET_ACCESS_KEY="$(terraform output -raw credential_secret_access_key)"
    ```

=== "Tofu"

    ```bash
    export AWS_ACCESS_KEY_ID="$(tofu output -raw credential_access_key)"
    export AWS_SECRET_ACCESS_KEY="$(tofu output -raw credential_secret_access_key)"
    ```

You can also persist these values in `set-env.sh` / `set-env.ps1` and source the file again before running the main infrastructure stack.

## 4. Review generated infrastructure values

Review `platform-configs/<cluster-name>/terraform/infrastructure/env.auto.tfvars`. Keep persistent changes in a separate override file as described in [Terraform value overrides](../2_concepts/overview_core_concept.md#terraform-value-overrides).

A few defaults that often need attention before the first apply:

- **`enable_cluster_public_endpoint = true`** binds a small EIP (`5_bgp`, 5 Mbit/s, traffic-charged) to the CCE master so the API server is reachable from the machine that runs Terraform. This is required for the in-stack Helm provider, for example the OpenBao Helm release, when applying from outside the VPC. After apply, the public IP is exposed as the `cluster_public_endpoint_ip` output. Set to `false` if you only run Terraform from inside the VPC, for example from a CI runner inside OTC, a bastion, or a VPN.
- **`enable_nat_gateway = true`** creates the NAT gateway for node egress. Adjust `nat_gateway_spec` and `nat_eip_bandwidth_size` if the default size does not fit your cluster.
- **`enable_shared_load_balancer = true`** creates the shared ELB used by Traefik. Set `enable_dedicated_load_balancer = true` only if you also need an independent dedicated ELB.
- **`enable_openbao = true`** rolls out the in-cluster OpenBao Helm release after CCE comes up. The release is not initialized or unsealed automatically. That happens manually in step 6 below.

## 5. Provision the CCE infrastructure

Proceed to:

```bash
cd ../infrastructure
```

The generated infrastructure backend stores state in:

```hcl
bucket = "bucket-tf-<cluster-name>-<stage>"
key    = "tf-state-<cluster-name>-<stage>"
```

The default backend endpoint is `https://obs.eu-de.otc.t-systems.com`, which is the technical OBS endpoint for the `eu-de` region. For `eu-nl`, adjust the generated backend endpoint and region before running `terraform init`.

Make sure `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` from the bootstrap stack are set in `set-env.sh` before initializing the backend.

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

This creates the DNS zone, required IAM agencies, networking, keypair, KMS keys, the CCE cluster, the kubara-managed StorageClasses, and the OpenBao Helm release.

The generated stack can optionally write a kubeconfig locally when `create_kubeconfig_local` is enabled.

## 6. OpenBao manual init and unseal

After the infrastructure apply has installed the OpenBao Helm release, OpenBao is running as a 3-replica HA Raft cluster but sealed. Until it is initialized and unsealed, the pods report `0/1` ready, which is expected because the readiness probe deliberately fails on a sealed pod.

The generated Helm release configures Raft peer discovery (`retry_join`) so the replicas find each other automatically. You only need to run `init` once on the first pod, then `unseal` on each replica.

### 1. Wait for the pods to be Running

```bash
kubectl -n openbao get pods -w
```

You should see `openbao-0`, `openbao-1`, and `openbao-2` reach `Running` state while still showing `0/1`. Press Ctrl-C once they all show `Running`.

### 2. Initialize OpenBao on the first pod

This generates the unseal keys and the initial root token. Run this exactly once. The output appears only once and cannot be recovered later.

```bash
kubectl exec -n openbao -ti openbao-0 -- bao operator init
```

Save the output immediately in an approved secure system that is accessible to the responsible team. Do not store unseal keys or the root token in Git, Terraform state, or shared plaintext files. By default OpenBao prints 5 unseal keys and 1 initial root token:

- Any 3 of the 5 unseal keys are needed to unseal a pod.
- Keep the keys under separate trusted operators or access controls where possible because anyone who collects 3 keys can decrypt the cluster.
- The root token gives full access, so rotate or revoke it after creating a long-lived admin role.

### 3. Unseal each pod with 3 keys

Unsealing is per-pod and per-restart. Repeat the command 3 times per pod, each time with a different unseal key:

```bash
# Pod 0
kubectl exec -n openbao -ti openbao-0 -- bao operator unseal <unseal-key-1>
kubectl exec -n openbao -ti openbao-0 -- bao operator unseal <unseal-key-2>
kubectl exec -n openbao -ti openbao-0 -- bao operator unseal <unseal-key-3>

# Pod 1
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

The Initial Root Token from step 2 is what the next stack, `openbao/`, uses to authenticate against OpenBao through the local port-forward. There is intentionally no Terraform output for it, so it is not stored in Terraform state.

The set-env script already contains commented-out lines for it. Uncomment them and fill in the token you saved:

```bash
# platform-configs/<cluster-name>/terraform/set-env.sh
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="hvb.AAAAAQ..."
```

Re-source the file so the new variables apply to the OpenBao Terraform layer:

```bash
source ../set-env.sh
```

### Pod restarts and auto-unseal

Until OpenBao supports auto-unseal with T Cloud Public KMS, every restarted OpenBao pod must be unsealed again with 3 of the 5 keys. This includes restarts caused by cluster upgrades, node maintenance, and OpenBao updates. The [T Cloud Public KMS wrapper](https://github.com/openbao/go-kms-wrapping/pull/63) has been merged, while the [OpenBao auto-unseal integration](https://github.com/openbao/openbao/issues/2302) is still pending.

Do not write OpenBao secrets in this first infrastructure apply. Apply OpenBao configuration and secrets in a separate step after OpenBao is initialized, unsealed, and a valid OpenBao token is available.

## 7. OpenBao configuration and secrets

kubara also renders a separate OpenBao Terraform layer:

```text
platform-configs/<cluster-name>/terraform/openbao
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

=== "Terraform"

    ```bash
    cd ../openbao
    terraform init
    terraform apply
    ```

=== "Tofu"

    ```bash
    cd ../openbao
    tofu init
    tofu apply
    ```

The layer configures a KV v2 mount, Kubernetes auth at `k8s-auth`, the namespace-scoped `k8s-kv-read` role and templated policy, the `external-secrets` role used only for the cluster-wide image pull secret, and the generated Grafana admin credentials.

User-provided secrets, the OAuth2 client credentials, `t-cloud-public-clouds-yaml` for ExternalDNS, and the Velero S3 credentials, are written through a separate `secrets.tf-example` file. Copy it to activate the blocks you need:

```bash
cp secrets.tf-example secrets.tf
```

Each block declares a `variable` and the matching `vault_kv_secret_v2` resource. The values come from `TF_VAR_*` environment variables in your sourced `set-env.sh`, which already has commented-out templates for each one, so no secret is ever written into a committed file. Delete the blocks you do not use before applying.

### Namespace-isolated secret access

T Cloud Public uses **namespace-isolated** secret access. Each consuming service reads only its own namespace's secrets:

| Secret | KV path | Consuming namespace |
|--------|---------|---------------------|
| Grafana admin / OAuth2 | `secret/<cluster>/<stage>/kube-prometheus-stack/*` | `kube-prometheus-stack` |
| Argo CD OAuth2 | `secret/<cluster>/<stage>/argocd/*` | `argocd` |
| OAuth2 Proxy | `secret/<cluster>/<stage>/oauth2-proxy/*` | `oauth2-proxy` |
| Velero S3 | `secret/<cluster>/<stage>/velero/*` | `velero` |
| ExternalDNS clouds.yaml | `secret/<cluster>/<stage>/external-dns/*` | `external-dns` |
| **Image pull secret** | `secret/<cluster>/<stage>/cluster_secrets/docker_config` | **all** (cluster-wide) |

Every chart renders a `SecretStore` in its own namespace that authenticates with that namespace's `default` ServiceAccount through the `k8s-kv-read` role. The templated policy matches the namespace segment in `secret/<cluster>/<stage>/<namespace>/*`, so a workload in one namespace cannot read another namespace's secrets. The image pull secret is the deliberate exception. It is distributed to every namespace through a `ClusterExternalSecret` and remains on the cluster-wide store under `cluster_secrets`.

Velero reads its S3 credentials through an `ExternalSecret` in the `velero` namespace. Keep the separate `external-secrets` service enabled because the Velero chart consumes External Secrets CRDs but does not install the External Secrets Operator itself. The generated `BackupStorageLocation` points at the same synchronized Kubernetes Secret, `velero-credentials`, key `cloud`.

With `services.velero.config.backupStorage.create: true`, the generated Velero values point at the Terraform-managed bucket name `velero-<cluster-name>-<stage>`. Set the matching `backupStorage.region` and `backupStorage.s3Url` values in the cluster config. If you use an existing OBS or S3-compatible bucket instead, set `backupStorage.create: false` and provide `backupStorage.bucketName`.

When Velero uses CSI snapshots, `backupMode: csi-snapshot` or `backupMode: csi-data-mover`, the generated values select the `t-cloud-public` `VolumeSnapshotClass` mapping for the CCE Everest CSI disk driver.

### OIDC admin access

The OpenBao Terraform layer can configure OIDC admin login. Put the overrides in `platform-configs/<cluster-name>/terraform/openbao/override.auto.tfvars`, not in the generated `env.auto.tfvars`; see [Terraform value overrides](../2_concepts/overview_core_concept.md#terraform-value-overrides). For example, use the following values for a Keycloak client:

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

## 8. Export kubeconfig and review outputs

Export the kubeconfig:

=== "Terraform"

    ```bash
    terraform output -raw kubeconfig > $HOME/.kube/kubara.yaml
    ```

=== "Tofu"

    ```bash
    tofu output -raw kubeconfig > $HOME/.kube/kubara.yaml
    ```

Keep this `kubara.yaml` local and do not commit it to Git.

Review the Terraform outputs:

=== "Terraform"

    ```bash
    terraform output
    ```

=== "Tofu"

    ```bash
    tofu output
    ```

The outputs are useful for the next steps around your cluster:

- `cluster_public_endpoint_ip` is the public IP for accessing the CCE control plane from outside the VPC
- `load_balancer_public_ip` is the public IP used by the shared load balancer for Kubernetes Services
- `dns_zone_name` and `dns_zone_masters` help when delegating or checking public DNS
- `storage_classes` shows the StorageClass names kubara created for the cluster

Now continue with the generic [Bootstrap Your Own Platform](../1_getting_started/bootstrapping.md) guide.
