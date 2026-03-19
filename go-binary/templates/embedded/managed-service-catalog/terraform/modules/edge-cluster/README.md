# STACKIT Edge Cloud Control Plane Module

Terraform module to provision a control plane node on STACKIT Edge Cloud (the server, not the cluster self).
This includes networking, security groups, a boot volume from an image, and a public IP.

## Usage

module "edgecloud_cluster_controlplane" {
source = "../modules/edge"

name               = "controlplane-${var.name}-${var.stage}"
project_id         = var.project_id
ipv4_prefix        = var.ipv4_prefix
edgecloud_image_id = var.edgecloud_image_id
controlplane       = {
flavor                   = "c1.4"
volume_size              = 30
volume_performance_class = "storage_premium_perf1"
}
}


<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >=1.9.3 |
| <a name="requirement_stackit"></a> [stackit](#requirement\_stackit) | 0.88.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_stackit"></a> [stackit](#provider\_stackit) | 0.88.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [stackit_network.edgecloud-network](https://registry.terraform.io/providers/stackitcloud/stackit/0.88.0/docs/resources/network) | resource |
| [stackit_network_interface.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.88.0/docs/resources/network_interface) | resource |
| [stackit_public_ip.public_ip](https://registry.terraform.io/providers/stackitcloud/stackit/0.88.0/docs/resources/public_ip) | resource |
| [stackit_security_group.public_ip_sec_group](https://registry.terraform.io/providers/stackitcloud/stackit/0.88.0/docs/resources/security_group) | resource |
| [stackit_security_group_rule.public_ip_sec_group_ingress_443](https://registry.terraform.io/providers/stackitcloud/stackit/0.88.0/docs/resources/security_group_rule) | resource |
| [stackit_security_group_rule.public_ip_sec_group_ingress_80](https://registry.terraform.io/providers/stackitcloud/stackit/0.88.0/docs/resources/security_group_rule) | resource |
| [stackit_server.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.88.0/docs/resources/server) | resource |
| [stackit_volume.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.88.0/docs/resources/volume) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_availability_zone"></a> [availability\_zone](#input\_availability\_zone) | Availability zone of the image. | `string` | `"eu01-1"` | no |
| <a name="input_controlplane"></a> [controlplane](#input\_controlplane) | Config for control plane nodes | <pre>object({<br/>    flavor                   = string<br/>    volume_size              = number<br/>    volume_performance_class = string<br/>  })</pre> | n/a | yes |
| <a name="input_edgecloud_image_id"></a> [edgecloud\_image\_id](#input\_edgecloud\_image\_id) | Edgecloud image ID. | `string` | n/a | yes |
| <a name="input_ipv4_nameservers"></a> [ipv4\_nameservers](#input\_ipv4\_nameservers) | IPv4 nameservers for the edgecloud network. | `list(string)` | <pre>[<br/>  "1.1.1.1",<br/>  "1.0.0.1",<br/>  "8.8.8.8"<br/>]</pre> | no |
| <a name="input_ipv4_prefix"></a> [ipv4\_prefix](#input\_ipv4\_prefix) | IPv4 prefix for the edgecloud network. | `string` | n/a | yes |
| <a name="input_labels"></a> [labels](#input\_labels) | Labels for the edgecloud cluster. | `map(string)` | <pre>{<br/>  "role": "controlplane"<br/>}</pre> | no |
| <a name="input_name"></a> [name](#input\_name) | Name of the edgecloud cluster. | `string` | n/a | yes |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | The StackIT ProjectID. | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_private_ip"></a> [private\_ip](#output\_private\_ip) | Private IP of the created Edge Cloud instance |
| <a name="output_public_ip"></a> [public\_ip](#output\_public\_ip) | Public IP of the created Edge Cloud instance |
| <a name="output_server_id"></a> [server\_id](#output\_server\_id) | ID of the created Edge Cloud instance |
<!-- END_TF_DOCS -->