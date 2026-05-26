output "instance_id" {
  description = "ID of the Edge Cloud instance (created or reused)."
  value       = var.create ? stackit_edgecloud_instance.this[0].instance_id : var.instance_id
}

output "frontend_url" {
  description = "Frontend URL of the created Edge Cloud instance. Null when reusing an existing instance."
  value       = var.create ? stackit_edgecloud_instance.this[0].frontend_url : null
}

output "status" {
  description = "Current status of the created Edge Cloud instance. Null when reusing an existing instance."
  value       = var.create ? stackit_edgecloud_instance.this[0].status : null
}
