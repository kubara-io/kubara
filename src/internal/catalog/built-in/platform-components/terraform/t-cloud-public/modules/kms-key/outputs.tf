output "id" {
  description = "KMS key ID."
  value       = opentelekomcloud_kms_key_v1.this.id
}

output "key_alias" {
  description = "KMS key alias."
  value       = opentelekomcloud_kms_key_v1.this.key_alias
}
