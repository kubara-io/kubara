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

## Configuration

Use these values in `config.yaml`:

```yaml
terraform:
  provider: stackit
  projectId: <project-id>
  kubernetesType: ske
  kubernetesVersion: 1.34
  dnsContactEmail: <email>
```

For STACKIT SKE, set `projectId` to the STACKIT project ID that should own the DNS zone, IAM resources, Secrets Manager, optional Velero bucket, and the SKE cluster.

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

=== "Tofu"

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

=== "Tofu"

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

===  "Tofu"

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

===  "Tofu"

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

=== "Tofu"

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

=== "Tofu"

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

=== "Tofu"

    ```bash
    tofu output vault_user_ro_password_b64
    ```

## 7. Optional: OAuth2-related Vault entries via Terraform

If you use OAuth2, create a GitHub application as shown [here](../4_building_your_platform/sso/add_sso.md).

If you want Terraform to create OAuth2-related Vault entries:

* Use `set-env.sh` / `set-env.ps1` for `TF_VAR_*` in `platform-configs/<cluster-name>/terraform/`
* `TF_Var_image_pull_secret` will already be set by kubara with what is present in the `.env`
* In `platform-configs/<cluster-name>/terraform/infrastructure`, copy `secrets.tf-oauth2` to `oauth2-secrets.tf` and adjust values if needed

Load the variables and apply:

=== "Terraform"

    ```bash
    cp secrets.tf-oauth2 oauth2-secrets.tf
    source ../set-env.sh
    # or for PowerShell
    # Copy-Item secrets.tf-oauth2 oauth2-secrets.tf
    . ..\set-env.ps1
    terraform apply
    ```

=== "Tofu"

    ```bash
    cp secrets.tf-oauth2 oauth2-secrets.tf
    source ../set-env.sh
    # or for PowerShell
    # Copy-Item secrets.tf-oauth2 oauth2-secrets.tf
    . ..\set-env.ps1
    tofu apply
    ```

!!! warning
     You need to set these environment variables again before re-applying Terraform if they are not persisted in your shell/session setup.

To clean up:

=== "Terraform"

    ```bash
    terraform state rm \
      vault_kv_secret_v2.image_pull_secret \
      vault_kv_secret_v2.oauth2_creds \
      vault_kv_secret_v2.argo_oauth2_creds \
      vault_kv_secret_v2.grafana_oauth2_creds \
      random_password.oauth2_cookie_secret
    ```

=== "Tofu"

    ```bash
    tofu state rm \
      vault_kv_secret_v2.image_pull_secret \
      vault_kv_secret_v2.oauth2_creds \
      vault_kv_secret_v2.argo_oauth2_creds \
      vault_kv_secret_v2.grafana_oauth2_creds \
      random_password.oauth2_cookie_secret
    ```

Now continue with the generic [Bootstrap Your Own Platform](../1_getting_started/bootstrapping.md) guide.
