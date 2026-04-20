# STACKIT Cloud and Edge setup

We recommend using Terraform to provisioning your Kubernetes and additional infrastructure like a Secret Manager instance.
For this purposes we provide the necessary terraform configuration and modules.

!!! warning
    In kubara version `0.2.0`, this step does not merge user-customized Terraform values and will overwrite existing Terraform files.

Generate Terraform modules:

```bash
kubara generate --terraform
```

Commit and push the generated files to your Git repository.

!!! info 
    You will need access to the STACKIT API. Setup instructions are available in the [Terraform provider documentation](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs) & [STACKIT Docs](https://docs.stackit.cloud/platform/access-and-identity/service-accounts/how-tos/manage-service-accounts/).
    Make sure your created Service Account has Project Owner permissions.

## 1. Terraform Bootstrap

Before the first `terraform init`, prepare and load your environment variables:

```bash
cd customer-service-catalog/terraform/<cluster-name>
cp set-env-changeme.sh set-env.sh
```
Set at least `STACKIT_SERVICE_ACCOUNT_KEY_PATH` in `set-env.sh` / `set-env.ps1` before sourcing.
```bash
source set-env.sh
# or for PowerShell
# cp set-env-changeme.ps1 set-env.ps1
# . .\set-env.ps1
```


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

## 2. Provisioning Infrastructure

!!! warning
    If you do not intend to use OAuth2 Proxy you can ignore some of the steps in the guide below pertaining to it, but might encounter some issues later.
    For more infos please look at our [FAQ](../../6_reference/faq.md#what-happens-when-oauth2-proxy-is-disabled).

Then proceed to:

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

===  "Terraform"

    ```bash
    terraform apply
    ```

===  "Tofu"

    ```bash
    tofu apply
    ```

This creates the Kubernetes cluster and all required infrastructure.

Export your kubeconfig:

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

Review Terraform outputs:

=== "Terraform"

    ```bash
    terraform output
    ```

=== "Tofu"

    ```bash
    tofu output
    ```

Use Terraform outputs to update values in `config.yaml` where needed (for example, on Edge: `privateLoadBalancerIP` and `publicLoadBalancerIP`).
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

## 3. STACKIT Edge-Specific Notes

The provisioning steps remain the same. The only difference lies in the Terraform output:

* You'll retrieve additional values like `privateLoadBalancerIP` and `publicLoadBalancerIP`
* These need to be added to `config.yaml`

You must manually create the Kubernetes cluster via the cloud portal. This will be automated in the future.

Now continue with the generic guide on the [Bootstrap Your Own Platform](../bootstrapping.md) page.

---

