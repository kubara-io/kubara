variable "bucket_name" {
  description = "OBS bucket name. Must be globally unique."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$", var.bucket_name))
    error_message = "bucket_name must use lowercase letters, numbers, and hyphens, start and end with an alphanumeric character, and be 3 to 63 characters long."
  }
}

variable "user_name" {
  description = "Identity user name for OBS access credentials."
  type        = string
}

variable "acl" {
  description = "OBS bucket ACL."
  type        = string
  default     = "private"
}

variable "versioning" {
  description = "Enable OBS bucket versioning."
  type        = bool
  default     = true
}

variable "parallel_fs" {
  description = "Create the bucket as a parallel file system."
  type        = bool
  default     = false
}

variable "enable_server_side_encryption" {
  description = "Enable OBS server-side encryption with KMS."
  type        = bool
  default     = true
}

variable "kms_key_id" {
  description = "KMS key ID used for OBS server-side encryption and the generated KMS access policy."
  type        = string
  default     = null
}
