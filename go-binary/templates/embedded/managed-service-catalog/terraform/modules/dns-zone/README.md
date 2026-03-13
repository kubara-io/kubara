# STACKIT DNS Zone Module

Terraform module to create a DNS zone in Stackit.
You have the option to choose between: .run.onstackit.cloud, .stackit.gg, .stackit.rocks, .stackit.run or .stackit.zone.
The DNS zone is managed by the Stackit DNS service.

## Usage
```
module "dns_zone" {
  source = "../modules/dns_zone"

  project_id = var.project_id
  name       = var.name
  dns_name   = var.dns_name
}
```
<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >=1.9.3 |
| <a name="requirement_stackit"></a> [stackit](#requirement\_stackit) | 0.87.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_stackit"></a> [stackit](#provider\_stackit) | 0.87.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [stackit_dns_zone.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.87.0/docs/resources/dns_zone) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_acl"></a> [acl](#input\_acl) | Access control list (CIDR), e.g. 0.0.0.0/0 | `string` | `"0.0.0.0/0"` | no |
| <a name="input_active"></a> [active](#input\_active) | Whether the zone is active | `bool` | `true` | no |
| <a name="input_contact_email"></a> [contact\_email](#input\_contact\_email) | Contact e-mail for the zone | `string` | `"hostmaster@stackit.cloud"` | no |
| <a name="input_default_ttl"></a> [default\_ttl](#input\_default\_ttl) | Default time to live in seconds | `number` | `300` | no |
| <a name="input_description"></a> [description](#input\_description) | DNS Zone for managing services | `string` | `"DNS Zone for managing services"` | no |
| <a name="input_dns_name"></a> [dns\_name](#input\_dns\_name) | DNS zone name (e.g. example). You have the option to choose between: .run.onstackit.cloud, .stackit.gg, .stackit.rocks, .stackit.run or .stackit.zone | `string` | n/a | yes |
| <a name="input_is_reverse_zone"></a> [is\_reverse\_zone](#input\_is\_reverse\_zone) | Whether this is a reverse lookup zone | `bool` | `false` | no |
| <a name="input_name"></a> [name](#input\_name) | User-defined name of the DNS Zone | `string` | n/a | yes |
| <a name="input_negative_cache"></a> [negative\_cache](#input\_negative\_cache) | Negative caching in seconds | `number` | `60` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | STACKIT project ID to associate the DNS zone with | `string` | n/a | yes |
| <a name="input_refresh_time"></a> [refresh\_time](#input\_refresh\_time) | Refresh time in seconds | `number` | `3600` | no |
| <a name="input_retry_time"></a> [retry\_time](#input\_retry\_time) | Retry time in seconds | `number` | `600` | no |
| <a name="input_type"></a> [type](#input\_type) | Zone type: primary or secondary | `string` | `"primary"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_id"></a> [id](#output\_id) | Internal Terraform resource ID (project\_id,zone\_id) |
| <a name="output_name"></a> [name](#output\_name) | Name of the DNS zone |
| <a name="output_primary_name_server"></a> [primary\_name\_server](#output\_primary\_name\_server) | FQDN of the primary name server |
| <a name="output_record_count"></a> [record\_count](#output\_record\_count) | Number of records in the zone |
| <a name="output_serial_number"></a> [serial\_number](#output\_serial\_number) | Serial number of the zone |
| <a name="output_state"></a> [state](#output\_state) | Current state (e.g. CREATE\_SUCCEEDED) |
| <a name="output_visibility"></a> [visibility](#output\_visibility) | Zone visibility (e.g. public) |
| <a name="output_zone_id"></a> [zone\_id](#output\_zone\_id) | Unique ID of the created DNS zone |
<!-- END_TF_DOCS -->