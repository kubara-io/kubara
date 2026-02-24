# STACKIT SKE Cluster Module

Terraform module to provision a fully managed Kubernetes cluster using STACKIT Kubernetes Engine (SKE).
Supports multiple node pools, custom labels, taints, availability zones, and version constraints.

## Usage
```
module "ske_cluster" {
  source = "./stackit_ske_cluster"

  project_id             = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
  name                   = "example-cluster"
  region                 = "eu01"         # optional, falls back to provider region
  kubernetes_version_min = "1.31.7"       # optional, uses latest if unset

  node_pools = [
    {
      name               = "np-frontend"
      machine_type       = "c1.2"
      minimum            = 2
      maximum            = 4
      availability_zones = ["eu01-1", "eu01-2"]
      labels             = { component = "frontend" }
      taints = [
        { key = "role", value = "frontend", effect = "NoSchedule" }
      ]
    },
    {
      name               = "np-backend"
      machine_type       = "c1.2"
      minimum            = 2
      maximum            = 5
      availability_zones = ["eu01-1", "eu01-2", "eu01-3"]
      labels             = { component = "backend" }
      taints             = []
    }
  ]
}
```
<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | >=1.9.3 |
| <a name="requirement_local"></a> [local](#requirement\_local) | 2.7.0 |
| <a name="requirement_stackit"></a> [stackit](#requirement\_stackit) | 0.81.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_local"></a> [local](#provider\_local) | 2.7.0 |
| <a name="provider_stackit"></a> [stackit](#provider\_stackit) | 0.81.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [local_file.kubeconfig](https://registry.terraform.io/providers/hashicorp/local/2.7.0/docs/resources/file) | resource |
| [stackit_ske_cluster.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.81.0/docs/resources/ske_cluster) | resource |
| [stackit_ske_kubeconfig.this](https://registry.terraform.io/providers/stackitcloud/stackit/0.81.0/docs/resources/ske_kubeconfig) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_create_kubeconfig_local"></a> [create\_kubeconfig\_local](#input\_create\_kubeconfig\_local) | If true, write the kubeconfig to a local file | `bool` | `false` | no |
| <a name="input_expiration"></a> [expiration](#input\_expiration) | Expiration time for the kubeconfig in seconds | `number` | `86400` | no |
| <a name="input_kubeconfig_path"></a> [kubeconfig\_path](#input\_kubeconfig\_path) | Filesystem path where the kubeconfig will be written if create\_kubeconfig\_local is true | `string` | `"~/.kube/config"` | no |
| <a name="input_kubernetes_version_min"></a> [kubernetes\_version\_min](#input\_kubernetes\_version\_min) | Minimum Kubernetes version (e.g. "1.31.7"); uses latest if unset | `string` | `"1.31.8"` | no |
| <a name="input_name"></a> [name](#input\_name) | Name of the SKE cluster. Can only container 11 characters | `string` | n/a | yes |
| <a name="input_node_pools"></a> [node\_pools](#input\_node\_pools) | List of node pools. Each element must be an object with:<br/>- name               = string<br/>- machine\_type       = string<br/>- os\_version\_min     = optional(string)<br/>- minimum            = number<br/>- maximum            = number<br/>- availability\_zones = list(string)<br/>- labels             = map(string)<br/>- taints             = list(object({<br/>    key    = string<br/>    value  = string<br/>    effect = string<br/>  })) | <pre>list(object({<br/>    name               = string<br/>    machine_type       = string<br/>    os_version_min     = optional(string)<br/>    minimum            = number<br/>    maximum            = number<br/>    availability_zones = list(string)<br/>    labels             = map(string)<br/>    taints = list(object({<br/>      key    = string<br/>      value  = string<br/>      effect = string<br/>    }))<br/>  }))</pre> | `[]` | no |
| <a name="input_project_id"></a> [project\_id](#input\_project\_id) | STACKIT project ID to associate the cluster with | `string` | n/a | yes |
| <a name="input_refresh"></a> [refresh](#input\_refresh) | Refresh token for the SKE cluster | `bool` | `true` | no |
| <a name="input_region"></a> [region](#input\_region) | Region for the cluster; falls back to provider region if null | `string` | `"eu01"` | no |
| <a name="input_ske_maintenance"></a> [ske\_maintenance](#input\_ske\_maintenance) | n/a | <pre>object({<br/>    enable_kubernetes_version_updates    = bool<br/>    enable_machine_image_version_updates = bool<br/>    start                                = string<br/>    end                                  = string<br/>  })</pre> | <pre>{<br/>  "enable_kubernetes_version_updates": true,<br/>  "enable_machine_image_version_updates": true,<br/>  "end": "02:00:00Z",<br/>  "start": "01:00:00Z"<br/>}</pre> | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_egress_address_ranges"></a> [egress\_address\_ranges](#output\_egress\_address\_ranges) | Outbound CIDR ranges for cluster workloads |
| <a name="output_id"></a> [id](#output\_id) | Terraform ID (project\_id,region,name) |
| <a name="output_kubeconfig_file"></a> [kubeconfig\_file](#output\_kubeconfig\_file) | Path to the written kubeconfig file |
| <a name="output_kubeconfig_raw"></a> [kubeconfig\_raw](#output\_kubeconfig\_raw) | Raw admin kubeconfig (short-lived, sensitive) |
| <a name="output_kubernetes_version_used"></a> [kubernetes\_version\_used](#output\_kubernetes\_version\_used) | Full Kubernetes version currently running |
| <a name="output_node_pools"></a> [node\_pools](#output\_node\_pools) | List of node\_pools as returned by the API (including any read-only fields) |
<!-- END_TF_DOCS -->
