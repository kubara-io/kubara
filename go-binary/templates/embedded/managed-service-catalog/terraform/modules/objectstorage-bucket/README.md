# STACKIT ObjectStorage Bucket Module

Terraform module to provision STACKIT ObjectStorage:

1. Credentials group
2. Bucket
3. Credential
4. Create a policy for the bucket (WARNING: After creation of the policy, you can just manage the bucket over the credentials group!!)

It ensures the credentials group is created before the bucket (to avoid background enablement races), then generates one credential for you.

## Usage
```
module "objectstorage_bucket" {
  source = "../modules/objectstorage-bucket"

  # Required
  project_id                 = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  credentials_group_name     = "my-creds-group"
  bucket_name                = "my-example-bucket"

  # Optional
  region                     = "eu01"                     # falls back to provider region
}
```

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >=1.9.3 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 6.12.0 |
| <a name="requirement_stackit"></a> [stackit](#requirement\_stackit) | 0.85.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_stackit"></a> [stackit](#provider\_stackit) | 0.85.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [stackit_objectstorage_bucket.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.85.0/docs/resources/objectstorage_bucket) | resource |
| [stackit_objectstorage_credential.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.85.0/docs/resources/objectstorage_credential) | resource |
| [stackit_objectstorage_credentials_group.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.85.0/docs/resources/objectstorage_credentials_group) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_bucket_name"></a> [bucket\_name](#input\_bucket\_name) | DNS-compatible name for the ObjectStorage bucket | `string` | n/a | yes |
| <a name="input_credentials_group_name"></a> [credentials\_group\_name](#input\_credentials\_group\_name) | Display name for the ObjectStorage credentials group | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | STACKIT Project ID to associate all ObjectStorage resources with | `string` | n/a | yes |
| <a name="input_region"></a> [region](#input\_region) | Region for all resources; defaults to provider region if unset | `string` | `"eu01"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_bucket_id"></a> [bucket\_id](#output\_bucket\_id) | Terraform internal ID (project\_id,bucket\_name) |
| <a name="output_bucket_name"></a> [bucket\_name](#output\_bucket\_name) | Name of the created bucket |
| <a name="output_bucket_url_path_style"></a> [bucket\_url\_path\_style](#output\_bucket\_url\_path\_style) | Bucket endpoint URL (path-style) |
| <a name="output_bucket_url_virtual_hosted_style"></a> [bucket\_url\_virtual\_hosted\_style](#output\_bucket\_url\_virtual\_hosted\_style) | Bucket endpoint URL (virtual-hosted style) |
| <a name="output_credential_access_key"></a> [credential\_access\_key](#output\_credential\_access\_key) | Access key for the credential |
| <a name="output_credential_id"></a> [credential\_id](#output\_credential\_id) | ID of the created credential |
| <a name="output_credential_secret_access_key"></a> [credential\_secret\_access\_key](#output\_credential\_secret\_access\_key) | Secret access key (sensitive) |
| <a name="output_credentials_group_id"></a> [credentials\_group\_id](#output\_credentials\_group\_id) | ID of the created credentials group |
| <a name="output_credentials_group_urn"></a> [credentials\_group\_urn](#output\_credentials\_group\_urn) | URN of the credentials group |
<!-- END_TF_DOCS -->