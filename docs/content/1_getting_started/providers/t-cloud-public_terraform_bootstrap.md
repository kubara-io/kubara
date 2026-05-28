# T Cloud Public Terraform Bootstrap

For T Cloud Public setups, kubara generates a Terraform bootstrap stack that creates the OBS bucket and S3-compatible credentials for the main Terraform state backend.

Generate Terraform modules:

```bash
kubara generate --terraform
```

Commit and push the generated files to your Git repository.

## 1. Prepare Environment Variables

Before the first `terraform init`, prepare and load your environment variables:

```bash
cd customer-service-catalog/terraform/<cluster-name>
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

Then load the file:

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

### OBS-to-KMS Agency

The bootstrap stack creates an IAM agency named `OBSAccessKMS` (delegated to the `op_svc_obs` service principal) so that OBS can use the generated KMS key for server-side bucket encryption. Without this agency, the bucket creation fails with `Status=403 Forbidden, Code=AccessDenied` the moment `server_side_encryption` is set.

The agency is tenant-scoped and only needs to exist once per T Cloud Public tenant. If your tenant already has it (for example created out-of-band or from a previous bootstrap of another cluster), disable the in-stack creation:

```hcl
create_obs_kms_agency = false
```

If you do not need server-side encryption at all for the state bucket, set:

```hcl
enable_bucket_server_side_encryption = false
```

The agency module is then skipped entirely.

### Apply

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

## 3. Continue With Provisioning

Next, continue with [Provisioning Infrastructure (CCE)](t-cloud-public_provisioning_cce.md) for `terraform.kubernetesType: cce`.
