# STACKIT Terraform Bootstrap

We recommend using Terraform to provision your Kubernetes and additional infrastructure like a Secret Manager instance.
For this purpose we provide the necessary Terraform configuration and modules.

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

## 1. Prepare Environment Variables

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

## 2. Create Terraform Backend State

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

## 3. Continue With Provisioning

Next steps:

- [Provisioning Infrastructure (SKE)](stackit_provisioning_ske.md)
- [Provisioning Infrastructure (Edge Cloud)](stackit_provisioning_edgecloud.md)
