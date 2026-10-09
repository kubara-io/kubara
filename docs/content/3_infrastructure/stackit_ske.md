# STACKIT SKE

This page combines the shared STACKIT Terraform bootstrap steps with the SKE-specific provisioning flow.

kubara's provided STACKIT SKE preset creates the infrastructure pieces that kubara expects later during bootstrap:

- a DNS zone for `external-dns`
- a Secrets Manager instance for `external-secrets`
- an IAM service account for provider access
- optionally an object storage bucket for Velero
- the SKE Kubernetes cluster itself

The kubara provider key is `stackit` and the Kubernetes type is `ske`.

!!! info
    You will need access to the STACKIT API. Setup instructions are available in the [Terraform provider documentation](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs) and [STACKIT Docs](https://docs.stackit.cloud/platform/access-and-identity/service-accounts/how-tos/manage-service-accounts/).
    Make sure your created Service Account has Project Owner permissions.

!!! warning
    If you do not intend to use OAuth2 Proxy you can ignore some of the steps below that talk about it, but you might run into later differences in the generated setup.
    For more info look at our [FAQ](../9_reference/faq.md#what-happens-when-oauth2-proxy-is-disabled).

## Sizing and Price
As a starting point for your sizing, the default configuration matches "S", which is benchmarked and described here:
[Scale and HA - Sizing](../2_concepts/scale_and_ha.md/#argo-cd-sizing-component-configuration-per-t-shirt-size)

A default kubara setup deployed on STACKIT Kubernetes Engine (SKE) costs around 580€/Month, running 24/7 (Last Update: September, 2026).

The measured components are:  
- Kubernetes Cluster  
3 Worker Node VMs (Type: g3i.4), Bootvolumes, Controlplane management fee  
- Object Storage Bucket  
- Secrets Manager Secrets (Out of the box: 8 Secrets)  
- DNS Zone  
- Loadbalancer 
- Public IPs 

## Configuration

Use these values in `config.yaml`:

```yaml
terraform:
  provider: stackit
  projectId: <project-id>
  kubernetesType: ske
  kubernetesVersion: "1.36"
  dns:
    name: <dns-name>
    email: <email>
```

For STACKIT SKE, set `projectId` to the STACKIT project ID that should own the DNS zone, IAM resources, Secrets Manager, optional Velero bucket, and the SKE cluster.

Use a supported minor version such as `"1.36"`; the STACKIT provider resolves the patch version.
List available versions with:

```bash
stackit ske options --kubernetes-versions --region eu01
```

Set `terraform.kubernetesVersion` in `config.yaml` and regenerate Terraform to prepare an upgrade.

## 1. Generate Terraform modules

```bash
kubara generate --terraform
```

Commit and push the generated files to your Git repository.

## 2. Prepare environment variables

Before the first `terraform init`, prepare and load your environment variables:

```bash
cd platform-configs/<cluster-name>/terraform
cp set-env-changeme.sh set-env.sh
```

Set at least `STACKIT_SERVICE_ACCOUNT_KEY_PATH` in `set-env.sh` / `set-env.ps1` before sourcing.

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

Run:

=== "Terraform"

    ```bash
    terraform init
    terraform plan
    terraform apply
    ```

=== "OpenTofu"

    ```bash
    tofu init
    tofu plan
    tofu apply
    ```

Use the output to configure Terraform backend credentials:

=== "Terraform"

    ```bash
    terraform output debug | grep -E "credential_access_key|credential_secret_access_key"
    ```

=== "OpenTofu"

    ```bash
    tofu output debug | grep -E "credential_access_key|credential_secret_access_key"
    ```

You can set `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` in `set-env.sh` / `set-env.ps1` and source the file again, or export them directly:

```bash
export AWS_ACCESS_KEY_ID="<credential_access_key from terraform output>"
export AWS_SECRET_ACCESS_KEY="<credential_secret_access_key from terraform output>"
```

## 4. Provision the SKE infrastructure

Proceed to:

```bash
cd ../infrastructure
```

Run:

===  "Terraform"

    ```bash
    terraform init
    terraform plan
    ```

===  "OpenTofu"

    ```bash
    tofu init
    tofu plan
    ```

Check the values generated in `env.auto.tfvars`, which is [automatically applied in your Terraform deployment.](https://developer.hashicorp.com/terraform/language/values/variables#assign-values-to-variables)

Apply:

===  "Terraform"

    ```bash
    terraform apply
    ```

===  "OpenTofu"

    ```bash
    tofu apply
    ```

This creates the Kubernetes cluster and all required infrastructure.

## 5. Export the kubeconfig

===  "Terraform"

    ```bash
    # change command accordingly to your needs. For example change the name of your kubeconfig, to not overwrite any files
    terraform output -raw kubeconfig > $HOME/.kube/kubara.yaml
    ```

=== "OpenTofu"

    ```bash
    # change command accordingly to your needs. For example change the name of your kubeconfig, to not overwrite any files
    tofu output -raw kubeconfig > $HOME/.kube/kubara.yaml
    ```

Keep this `kubara.yaml` local and do not commit it to Git.

## 6. Review Terraform outputs

=== "Terraform"

    ```bash
    terraform output
    ```

=== "OpenTofu"

    ```bash
    tofu output
    ```

Use Terraform outputs to update values in `config.yaml` where needed.
Do **not** export Secrets Manager credentials into `.env`; these provider-specific `.env` variables were removed.

Sensitive output example:

=== "Terraform"

    ```bash
    terraform output vault_user_ro_password_b64
    ```

=== "OpenTofu"

    ```bash
    tofu output vault_user_ro_password_b64
    ```

## 7. Optional: OAuth2-related Vault entries via Terraform

If you use OAuth2 with GitHub, create a GitHub application as shown [here](../4_building_your_platform/sso/add_sso_github.md).

The optional `secrets.tf-oauth2` example manages registry credentials and the OAuth2 credentials for
OAuth2 Proxy, Argo CD and Grafana. With `terraform.ephemeralSecrets: true`, it uses ephemeral
inputs and write-only Vault payloads, so these
secret values are not saved in new Terraform/OpenTofu state or plan files. The Vault paths and
configuration revision counters remain in state: the entries are still managed.

The following workflow applies when `terraform.ephemeralSecrets: true`. If the field is omitted
or `false`, the previous stateful behavior is retained; migration is optional. See
[the configuration option](../2_concepts/overview_core_concept.md#ephemeral-secret-management).

**Required: Terraform/OpenTofu 1.11+ in both modes** and the catalog-pinned providers.
For this workflow, verify that `secrets.tf-oauth2` uses `data_json_wo`; `data_json` stores secrets in state.

### Create new entries

1. In `platform-configs/<cluster-name>/terraform/infrastructure`, copy `secrets.tf-oauth2` to
   `oauth2-secrets.tf`. Remove the resource and input-variable blocks for services you do not use.
2. Supply the matching `TF_VAR_*` inputs using your local `set-env.sh` / `set-env.ps1` or your CI
   secret store. The registry input, `TF_VAR_image_pull_secret`, is already rendered from `.env`.
   Keep these files local and out of Git; ephemeral values do not protect the source files.
3. Load the inputs and apply:

=== "Terraform"

    ```bash
    source ../set-env.sh
    terraform plan
    terraform apply
    ```

=== "OpenTofu"

    ```bash
    source ../set-env.sh
    tofu plan
    tofu apply
    ```

In PowerShell, use `Copy-Item secrets.tf-oauth2 oauth2-secrets.tf` and `. ..\set-env.ps1` instead
of the shell copy/source commands above.

Keep `oauth2-secrets.tf` enabled after the apply. **Do not run `state rm` or comment out its
resources as a cleanup step.** Removing managed Vault resources from configuration would plan to
delete the entries. To deliberately hand ownership to another system, use a separate, reviewed
state-removal procedure and remove the corresponding configuration together.

Supply the ephemeral inputs for **every plan and apply**, including when applying a saved plan;
they are intentionally absent from that plan. Unchanged applies leave the stored payloads alone,
even though the ephemeral cookie-password generator produces a fresh candidate on each run.

### Update or rotate a secret

The `oauth2_secret_versions` object controls writes independently for each entry. Its numbers are
configuration revisions, not Vault KV version numbers. Store the object in a non-secret
`oauth2-secrets.auto.tfvars` file and increment only the entry you want to update:

```hcl
oauth2_secret_versions = {
  image_pull_secret    = 1
  oauth2_creds         = 1
  argo_oauth2_creds    = 2 # write the new Argo CD credentials
  grafana_oauth2_creds = 1
}
```

Supply the new secret input before planning and applying. Changing an input alone, including a
client ID, does not update the write-only payload: increment its revision too. OpenTofu/Terraform
cannot compare the secret contents for drift because it does not retain a reference value.

Incrementing `oauth2_creds` also generates a new cookie secret and invalidates existing OAuth2
Proxy sessions. Recreating a deleted OAuth2 Proxy Vault entry also generates a new cookie secret.
After a rotation, verify ExternalSecret synchronization and restart consumers that read credentials
only at startup.

### Migrate existing entries

Set `terraform.ephemeralSecrets: true` in the cluster configuration and regenerate with
`kubara generate --terraform`. Replace the active `.tf` copy of the optional secret example
with the newly generated `secrets.tf-oauth2`, retaining any deliberate service exclusions.
Do not leave both old and new active copies. Review all infrastructure changes before applying.

Do not apply the new example as a fresh create over existing Vault paths. First determine whether
the entries are still managed (`terraform state list` / `tofu state list`) or were removed using
the old guide's `state rm` step.

1. Keep the existing Vault resource addresses, mount and paths. Replace the old example with the
   write-only version and use the catalog's pinned providers. The obsolete `vault.secondary`
   configuration is no longer needed; this example uses the root's default Vault provider.
2. Supply the **current** registry/client credentials from Vault. Do not commit them. The first
   write-only apply generates a new OAuth2 Proxy cookie secret, so existing sessions become invalid
   when the proxy picks it up and users must sign in again. This is a one-time migration effect;
   later applies retain the stored cookie unless `oauth2_secret_versions.oauth2_creds` changes.
3. If an entry is no longer in state, import it before applying. For example:

    ```bash
    tofu import vault_kv_secret_v2.oauth2_creds \
      '<mount>/data/<cluster-name>/<stage>/oauth2-proxy/oauth2_credentials'
    ```

    Use `terraform import` for Terraform. Repeat for the other existing entries:

    | Resource address | Path below `<mount>/data/<cluster-name>/<stage>/` |
    |---|---|
    | `vault_kv_secret_v2.image_pull_secret` | `cluster_secrets/docker_config` |
    | `vault_kv_secret_v2.argo_oauth2_creds` | `argocd/argo_oauth2_credentials` |
    | `vault_kv_secret_v2.grafana_oauth2_creds` | `kube-prometheus-stack/grafana_oauth2_credentials` |

4. Review the plan. Expect in-place writes with the initial write-only revisions, retaining the
   registry/client credentials and replacing the OAuth2 Proxy cookie secret. There should be no
   Vault-entry replacements or deletions. If the old managed
   `random_password.oauth2_cookie_secret` is still in state, its removal is expected: it only
   forgets the generated value and does not delete anything in Vault.
5. Apply, verify that registry/client credentials are unchanged and that the current state no
   longer contains secret payloads. Synchronize the new cookie secret to OAuth2 Proxy and verify
   a fresh sign-in. Check that the next plan is empty and a subsequent apply retains the cookie.

The first migration plan can still contain secrets from the old state. Protect it along with old
plan files, state backups and backend state versions; migration does not erase those historical
copies. STACKIT-generated cloud credentials still remain in their producing resources and
their automatically synchronized Vault entries.

### Grafana admin credentials

With the option enabled, the infrastructure root writes Grafana admin credentials using a
write-only payload. `grafana_admin_password` is ephemeral; when empty, a
password is generated. Ordinary applies retain the stored password. Increment
`grafana_admin_credentials_version` (default `1`) to write a new username/password pair.
Read the credentials from the `kube-prometheus-stack/grafana_credentials` Vault entry.

For an existing installation, supply its current password through `TF_VAR_grafana_admin_password`
for the migration plan and apply. Keep its username unchanged. The old managed random password
can then leave state without changing the login. Updating the Vault entry does not reset the
password in an initialized Grafana database; coordinate any later rotation with Grafana itself.
Historical state copies and migration plans can still contain the old password.

Now continue with the generic [Bootstrap Your Own Platform](../1_getting_started/bootstrapping.md) guide.
