# STACKIT Provisioning Infrastructure (SKE)

!!! warning
    If you do not intend to use OAuth2 Proxy you can ignore some of the steps in the guide below pertaining to it, but might encounter some issues later.
    For more infos please look at our [FAQ](../../6_reference/faq.md#what-happens-when-oauth2-proxy-is-disabled).

Proceed to:

```bash
cd customer-service-catalog/terraform/<cluster-name>/infrastructure
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

## Export kubeconfig

===  "Terraform"

    ```bash
    # change command accordingly to your needs. For example change the name of your kubeconfig, to not overwrite any files
    terraform output -raw kubeconfig_raw > $HOME/.kube/kubara.yaml
    ```

=== "Tofu"

    ```bash
    # change command accordingly to your needs. For example change the name of your kubeconfig, to not overwrite any files
    tofu output -raw kubeconfig_raw > $HOME/.kube/kubara.yaml
    ```

Keep this `kubara.yaml` local and do not commit it to Git.

## Review Terraform outputs

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

## Optional: OAuth2-related Vault entries via Terraform

If you use OAuth2, create a GitHub application as shown [here](/2_managing_your_platform/add_sso.md).

If you want Terraform to create OAuth2-related Vault entries:

* Use `set-env.sh` / `set-env.ps1` for `TF_VAR_*` in `customer-service-catalog/terraform/<cluster-name>/`
* `TF_Var_image_pull_secret` will already be set by kubara with what is present in the .env
* In `customer-service-catalog/terraform/<cluster-name>/infrastructure`, copy `secrets.tf-example` to `oauth2-secrets.tf` and adjust values if needed

Load the variables and apply:

=== "Terraform"

    ```bash
    cp secrets.tf-example oauth2-secrets.tf
    source ../set-env.sh
    # or for PowerShell
    # Copy-Item secrets.tf-example oauth2-secrets.tf
    . ..\set-env.ps1
    terraform apply
    ```

=== "Tofu"

    ```bash
    cp secrets.tf-example oauth2-secrets.tf
    source ../set-env.sh
    # or for PowerShell
    # Copy-Item secrets.tf-example oauth2-secrets.tf
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

Now continue with the generic guide on the [Bootstrap Your Own Platform](../bootstrapping.md) page.
