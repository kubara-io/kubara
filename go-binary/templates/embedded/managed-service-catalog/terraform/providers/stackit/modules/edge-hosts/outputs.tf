output "network_id" {
  description = "ID of the shared edge host network."
  value       = stackit_network.this.network_id
}

output "security_group_id" {
  description = "ID of the shared edge host security group."
  value       = stackit_security_group.this.security_group_id
}

output "host_metadata" {
  description = "Per-host metadata including IDs, role, and assigned IP addresses."
  value = {
    for name, node in local.nodes_by_name : name => {
      role                 = node.role
      availability_zone    = node.availability_zone
      server_id            = stackit_server.this[name].server_id
      volume_id            = stackit_volume.this[name].volume_id
      network_interface_id = stackit_network_interface.this[name].network_interface_id
      private_ip           = stackit_network_interface.this[name].ipv4
      public_ip            = try(stackit_public_ip.this[name].ip, null)
    }
  }
}
