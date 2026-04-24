output "server_id" {
  description = "ID of the created Edge Cloud instance"
  value       = stackit_server.this.server_id
}

output "private_ip" {
  description = "Private IP of the created Edge Cloud instance"
  value       = stackit_network_interface.this.ipv4
}

output "public_ip" {
  description = "Public IP of the created Edge Cloud instance"
  value       = stackit_public_ip.public_ip.ip
}
