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
  image_id            = module.edge_image.image_id
  network_name        = "edge-demo-network"
  security_group_name = "edge-demo-sg"
  ipv4_prefix         = "10.0.50.0/24"

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
