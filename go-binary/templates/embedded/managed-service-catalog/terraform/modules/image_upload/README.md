# STACKIT Edge Cloud Image Upload Module

Terraform module to upload a local image to STACKIT Edge Cloud.
The image can then be used to launch virtual machines on the Edge Cloud platform.

## Usage

module "edgecloud_image" {
source = "../modules/image_upload"

project_id      = var.project_id
name            = "talos-edge-image"
local_file_path = "./talos.raw"
min_disk_size   = 10
config = {
operating_system         = "linux"
operating_system_distro  = "talos"
operating_system_version = "v1.9.5"
}
}

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >=1.9.3 |
| <a name="requirement_stackit"></a> [stackit](#requirement\_stackit) | 0.86.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_stackit"></a> [stackit](#provider\_stackit) | 0.86.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [stackit_image.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.86.0/docs/resources/image) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_config"></a> [config](#input\_config) | Configuration of the image. | <pre>object({<br/>    operating_system         = string<br/>    operating_system_distro  = string<br/>    operating_system_version = string<br/>  })</pre> | <pre>{<br/>  "operating_system": "linux",<br/>  "operating_system_distro": "talos",<br/>  "operating_system_version": "v1.9.5"<br/>}</pre> | no |
| <a name="input_disk_format"></a> [disk\_format](#input\_disk\_format) | Disk format of the image. | `string` | `"raw"` | no |
| <a name="input_local_file_path"></a> [local\_file\_path](#input\_local\_file\_path) | Local file path of the image. | `string` | n/a | yes |
| <a name="input_min_disk_size"></a> [min\_disk\_size](#input\_min\_disk\_size) | Minimum disk size of the image. | `number` | `10` | no |
| <a name="input_name"></a> [name](#input\_name) | Name of the image. | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The STACKIT ProjectID. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_image_id"></a> [image\_id](#output\_image\_id) | ID of the created image. |
<!-- END_TF_DOCS -->