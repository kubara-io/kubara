output "id" {
  description = "CCE cluster ID."
  value       = opentelekomcloud_cce_cluster_v3.this.id
}

output "node_pools" {
  description = "CCE node pools created for the cluster."
  value       = opentelekomcloud_cce_node_pool_v3.this
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
