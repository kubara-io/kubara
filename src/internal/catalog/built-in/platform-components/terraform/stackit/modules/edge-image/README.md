# STACKIT Edge Image Module

Terraform module to upload a local Edge image artifact to STACKIT.

## Usage

```hcl
module "edge_image" {
  source = "../modules/edge-image"

  project_id      = var.project_id
  name            = "talos-edge-image"
  local_file_path = "./talos.raw"
  min_disk_size   = 30

  operating_system         = "linux"
  operating_system_distro  = "talos"
  operating_system_version = "v1.12.5-stackit.v1.7.1"
}
```

## Outputs

- `image_id`: ID of the uploaded image
