output "id" {
  description = "CCE cluster ID."
  value       = opentelekomcloud_cce_cluster_v3.this.id
}

output "public_endpoint_ip" {
  description = "Public IPv4 address bound to the CCE master, if enable_public_endpoint is true."
  value       = var.enable_public_endpoint ? opentelekomcloud_vpc_eip_v1.public_endpoint[0].publicip[0].ip_address : null
}

output "public_endpoint_eip_id" {
  description = "EIP resource ID bound to the CCE master, if enable_public_endpoint is true."
  value       = var.enable_public_endpoint ? opentelekomcloud_vpc_eip_v1.public_endpoint[0].id : null
}

output "node_pools" {
  description = "CCE node pools created for the cluster."
  value       = opentelekomcloud_cce_node_pool_v3.this
}

output "addons" {
  description = "CCE addons managed for the cluster."
  value       = opentelekomcloud_cce_addon_v3.this
}

output "kubeconfig_raw" {
  description = "Raw admin kubeconfig."
  value       = data.opentelekomcloud_cce_cluster_kubeconfig_v3.this.kubeconfig
  sensitive   = true
}

output "kubeconfig_file" {
  description = "Path to the written kubeconfig file."
  value       = try(local_file.kubeconfig[0].filename, "")
}
