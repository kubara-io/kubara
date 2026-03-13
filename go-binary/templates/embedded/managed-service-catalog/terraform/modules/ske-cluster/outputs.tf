output "id" {
  description = "Terraform ID (project_id,region,name)"
  value       = stackit_ske_cluster.this.id
}

output "kubernetes_version_used" {
  description = "Full Kubernetes version currently running"
  value       = stackit_ske_cluster.this.kubernetes_version_used
}

output "egress_address_ranges" {
  description = "Outbound CIDR ranges for cluster workloads"
  value       = stackit_ske_cluster.this.egress_address_ranges
}

output "node_pools" {
  description = "List of node_pools as returned by the API (including any read-only fields)"
  value       = stackit_ske_cluster.this.node_pools
}



### Kubeconfig
output "kubeconfig_raw" {
  description = "Raw admin kubeconfig (short-lived, sensitive)"
  value       = stackit_ske_kubeconfig.this.kube_config
  sensitive   = true
}

output "kubeconfig_file" {
  description = "Path to the written kubeconfig file"
  value       = try(local_file.kubeconfig[0].filename, "")
}
