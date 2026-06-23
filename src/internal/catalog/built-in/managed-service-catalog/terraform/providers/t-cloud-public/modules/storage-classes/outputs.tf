output "names" {
  description = "Created StorageClass names."
  value       = keys(kubernetes_storage_class_v1.this)
}
