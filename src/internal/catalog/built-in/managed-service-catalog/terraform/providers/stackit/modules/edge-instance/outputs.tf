output "instance_id" {
  description = "ID of the Edge Cloud instance."
  value       = stackit_edgecloud_instance.this.instance_id
}

output "frontend_url" {
  description = "Frontend URL of the created Edge Cloud instance."
  value       = stackit_edgecloud_instance.this.frontend_url
}

output "status" {
  description = "Current status of the created Edge Cloud instance."
  value       = stackit_edgecloud_instance.this.status
}

output "kubeconfig" {
  description = "Edge Cloud kubeconfig (short-lived, sensitive)."
  value       = stackit_edgecloud_kubeconfig.this.kubeconfig
  sensitive   = true
}
