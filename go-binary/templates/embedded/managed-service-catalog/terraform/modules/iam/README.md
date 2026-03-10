# STACKIT IAM Module

Creates a STACKIT Service Account and an Servivce Account Key, with optional rotation trigger.

## Usage

module "service_account" {
source = "../modules/iam"

project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
name       = "sa01"

# Optional: override default TTL
ttl_days   = 180

}

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >=1.9.3 |
| <a name="requirement_stackit"></a> [stackit](#requirement\_stackit) | 0.86.0 |
| <a name="requirement_time"></a> [time](#requirement\_time) | 0.13.1 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_stackit"></a> [stackit](#provider\_stackit) | 0.86.0 |
| <a name="provider_time"></a> [time](#provider\_time) | 0.13.1 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [stackit_authorization_project_role_assignment.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.86.0/docs/resources/authorization_project_role_assignment) | resource |
| [stackit_service_account.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.86.0/docs/resources/service_account) | resource |
| [stackit_service_account_key.no_ttl](https://registry.terraform.io/providers/stackitcloud/stackit/0.86.0/docs/resources/service_account_key) | resource |
| [stackit_service_account_key.with_ttl](https://registry.terraform.io/providers/stackitcloud/stackit/0.86.0/docs/resources/service_account_key) | resource |
| [time_rotating.rotate](https://registry.terraform.io/providers/hashicorp/time/0.13.1/docs/resources/rotating) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_name"></a> [name](#input\_name) | Name of the service account (max 30 chars; lowercase alphanumeric and hyphens only) | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | STACKIT project ID where the service account will be created | `string` | n/a | yes |
| <a name="input_role_assignment_role"></a> [role\_assignment\_role](#input\_role\_assignment\_role) | The name of the role to assign (e.g. owner, editor, viewer). | `string` | n/a | yes |
| <a name="input_sa_key_ttl_days"></a> [sa\_key\_ttl\_days](#input\_sa\_key\_ttl\_days) | Service account key TTL in days. If null, TTL and key rotation are disabled. | `number` | `null` | no |
| <a name="input_sa_key_ttl_rotation_buffer_days"></a> [sa\_key\_ttl\_rotation\_buffer\_days](#input\_sa\_key\_ttl\_rotation\_buffer\_days) | Number of days before TTL expiration when the key rotation should be triggered. Must be less than sa\_key\_ttl\_days. | `number` | `10` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_role_assignment_id"></a> [role\_assignment\_id](#output\_role\_assignment\_id) | The ID of the created project role assignment. |
| <a name="output_role_assignment_resource_id"></a> [role\_assignment\_resource\_id](#output\_role\_assignment\_resource\_id) | The resource ID to which the role was applied. |
| <a name="output_role_assignment_role"></a> [role\_assignment\_role](#output\_role\_assignment\_role) | The role that was assigned. |
| <a name="output_role_assignment_subject"></a> [role\_assignment\_subject](#output\_role\_assignment\_subject) | The subject (user/service account/client) that received the role. |
| <a name="output_service_account_email"></a> [service\_account\_email](#output\_service\_account\_email) | Email address of the service account |
| <a name="output_service_account_id"></a> [service\_account\_id](#output\_service\_account\_id) | Internal ID of the service account (project\_id,email) |
| <a name="output_service_account_key_id"></a> [service\_account\_key\_id](#output\_service\_account\_key\_id) | Internal ID of the service account key |
| <a name="output_service_account_key_json"></a> [service\_account\_key\_json](#output\_service\_account\_key\_json) | Service account key JSON (sensitive) |
<!-- END_TF_DOCS -->