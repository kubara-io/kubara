# STACKIT Edge Instance Module

Terraform module for creating a STACKIT Edge Cloud instance.

## What this module does

- Creates one `stackit_edgecloud_instance`
- Reads the available Edge Cloud plan from the STACKIT API

## Usage

The STACKIT provider must have beta resources enabled.
The generated kubara infrastructure template already sets this.

```hcl
module "edge_instance" {
  source = "../modules/edge-instance"

  project_id   = var.project_id
  display_name = "edge1234"
  region       = "eu01"
  description  = "kubara edge instance"
  expiration   = 86400
}
```

## Outputs

- `instance_id`: Edge Cloud instance ID
- `frontend_url`: Edge frontend URL
- `status`: Instance status
- `kubeconfig_raw`: Raw Edge Cloud kubeconfig (short-lived, sensitive)
