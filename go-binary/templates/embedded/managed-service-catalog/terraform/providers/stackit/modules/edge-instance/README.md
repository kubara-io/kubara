# STACKIT Edge Instance Module

Terraform module for creating or reusing a STACKIT Edge Cloud instance.

## What this module does

- Creates `stackit_edgecloud_instance` when `create = true`
- Reuses an externally provided `instance_id` when `create = false`

## Usage

```hcl
module "edge_instance" {
  source = "../modules/edge-instance"

  project_id   = var.project_id
  create       = true
  display_name = "edge1234"
  plan_name    = "preview"
  region       = "eu01"
  description  = "kubara edge instance"
}
```

## Outputs

- `instance_id`: Edge Cloud instance ID
- `frontend_url`: Edge frontend URL (null for reused instances)
- `status`: Instance status (null for reused instances)
