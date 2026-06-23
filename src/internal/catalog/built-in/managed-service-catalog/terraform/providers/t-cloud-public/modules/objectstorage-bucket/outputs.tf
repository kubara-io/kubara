output "bucket_name" {
  description = "OBS bucket name."
  value       = opentelekomcloud_obs_bucket.this.bucket
}

output "bucket_id" {
  description = "OBS bucket Terraform ID."
  value       = opentelekomcloud_obs_bucket.this.id
}

output "user_id" {
  description = "Identity user ID used for OBS access."
  value       = opentelekomcloud_identity_user_v3.this.id
}

output "obs_only_group_id" {
  description = "Identity group ID used as the base OBS-only user group."
  value       = opentelekomcloud_identity_group_v3.obs_only.id
}

output "kms_access_group_id" {
  description = "Identity group ID used for OBS KMS access, if server-side encryption is enabled."
  value       = var.enable_server_side_encryption ? opentelekomcloud_identity_group_v3.kms_access[0].id : null
}

output "kms_access_role_id" {
  description = "Identity role ID used for OBS KMS access, if server-side encryption is enabled."
  value       = var.enable_server_side_encryption ? opentelekomcloud_identity_role_v3.kms_access[0].id : null
}

output "credential_access_key" {
  description = "Access key for OBS/S3 access."
  value       = opentelekomcloud_identity_credential_v3.this.access
}

output "credential_secret_access_key" {
  description = "Secret key for OBS/S3 access."
  value       = opentelekomcloud_identity_credential_v3.this.secret
  sensitive   = true
}
