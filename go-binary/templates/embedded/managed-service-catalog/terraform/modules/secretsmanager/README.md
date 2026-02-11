# STACKIT Secrets Manager Module

Terraform module to provision a STACKIT Secrets Manager instance with one or more users.  
Supports configuring user access (read-only or read-write) for secure secret storage and management.

## Usage

module "secretsmgr" {
source = "./stackit_secretsmanager"

project_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
name       = "example-instance"

users = [
{
description   = "vault-user-rw"
write_enabled = true
},
{
description   = "vault-user-ro"
write_enabled = false
}
]
}
<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >=1.9.3 |
| <a name="requirement_stackit"></a> [stackit](#requirement\_stackit) | 0.80.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_stackit"></a> [stackit](#provider\_stackit) | 0.80.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [stackit_secretsmanager_instance.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.80.0/docs/resources/secretsmanager_instance) | resource |
| [stackit_secretsmanager_user.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.80.0/docs/resources/secretsmanager_user) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_acls"></a> [acls](#input\_acls) | Set of CIDR blocks allowed to access this instance | `set(string)` | <pre>[<br/>  "0.0.0.0/0"<br/>]</pre> | no |
| <a name="input_name"></a> [name](#input\_name) | Name of the Secrets Manager instance | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | STACKIT project ID | `string` | n/a | yes |
| <a name="input_users"></a> [users](#input\_users) | List of users to create.<br/>Each element must be an object with:<br/>- description (string): human‐readable ID for the user<br/>- write\_enabled (bool): whether this user has write access | <pre>list(object({<br/>    description   = string<br/>    write_enabled = bool<br/>  }))</pre> | `[]` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_instance"></a> [instance](#output\_instance) | Map of instance attributes |
| <a name="output_instance_id"></a> [instance\_id](#output\_instance\_id) | ID of the created Secrets Manager instance |
| <a name="output_users"></a> [users](#output\_users) | Map of created users, keyed by description |
<!-- END_TF_DOCS -->