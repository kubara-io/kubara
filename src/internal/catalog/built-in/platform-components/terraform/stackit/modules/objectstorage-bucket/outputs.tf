# Credentials Group
output "credentials_group_id" {
  description = "ID of the created credentials group"
  value       = stackit_objectstorage_credentials_group.this.credentials_group_id
}

output "credentials_group_urn" {
  description = "URN of the credentials group"
  value       = stackit_objectstorage_credentials_group.this.urn
}

# Bucket
output "bucket_id" {
  description = "Terraform internal ID (project_id,bucket_name)"
  value       = stackit_objectstorage_bucket.this.id
}

output "bucket_name" {
  description = "Name of the created bucket"
  value       = stackit_objectstorage_bucket.this.name
}

output "bucket_url_path_style" {
  description = "Bucket endpoint URL (path-style)"
  value       = stackit_objectstorage_bucket.this.url_path_style
}

output "bucket_url_virtual_hosted_style" {
  description = "Bucket endpoint URL (virtual-hosted style)"
  value       = stackit_objectstorage_bucket.this.url_virtual_hosted_style
}

# output "backend_config" {
#   value = <<EOT
#     backend "s3" {
#       bucket                      = "${stackit_objectstorage_bucket.this.name}"
#       key                         = "!!!<change-me>!!!"
#       endpoints                   = {
#         s3 = "https://object.storage.eu01.onstackit.cloud"
#       }
#       region                      = "eu01"
#       skip_credentials_validation = true
#       skip_region_validation      = true
#       skip_s3_checksum            = true
#       skip_requesting_account_id  = true
#     }
#   EOT
# }

# Credentials
output "credential_id" {
  description = "ID of the created credential"
  value       = stackit_objectstorage_credential.this.credential_id
}

output "credential_access_key" {
  description = "Access key for the credential"
  value       = stackit_objectstorage_credential.this.access_key
}

output "credential_secret_access_key" {
  description = "Secret access key (sensitive)"
  value       = stackit_objectstorage_credential.this.secret_access_key
  sensitive   = true
}
