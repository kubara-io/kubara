# STACKIT Edge Hosts Module

Terraform module to provision edge host infrastructure for one cluster:

- shared network
- shared security group (+ ingress rules)
- one volume, NIC, and VM per node
- optional public IP per node

## Usage

```hcl
module "edge_hosts" {
  source = "../modules/edge-hosts"

  name                = "edge-demo"
  project_id          = var.project_id
  # Use an existing image ID, e.g. from STACKIT project image list.
  # You can also use `terraform output edge_uploaded_image_id`
  # after running the optional edge_image upload module.
  image_id            = "11111111-2222-3333-4444-555555555555"
  network_name        = "edge-demo-network"
  security_group_name = "edge-demo-sg"
  ipv4_prefix         = "10.0.50.0"

  nodes = [
    {
      name                     = "edge-demo-cp-1"
      role                     = "controlplane"
      flavor                   = "g2i.8"
      volume_size              = 30
      volume_performance_class = "storage_premium_perf1"
      availability_zone        = "eu01-1"
      assign_public_ip         = true
      labels                   = {}
    }
  ]
}
```

## Outputs

- `network_id`: shared network ID
- `security_group_id`: shared security group ID
- `host_metadata`: per-node IDs and IPs
