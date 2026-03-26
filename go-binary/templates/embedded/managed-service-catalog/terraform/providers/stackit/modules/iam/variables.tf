variable "project_id" {
  description = "STACKIT project ID where the service account will be created"
  type        = string
}

variable "name" {
  description = "Name of the service account (max 30 chars; lowercase alphanumeric and hyphens only)"
  type        = string

  validation {
    condition     = length(var.name) <= 30
    error_message = "Service account name must not exceed 30 characters."
  }
}

variable "sa_key_ttl_days" {
  description = "Service account key TTL in days. If null, TTL and key rotation are disabled."
  type        = number
  default     = null
}

variable "sa_key_ttl_rotation_buffer_days" {
  description = "Number of days before TTL expiration when the key rotation should be triggered. Must be less than sa_key_ttl_days."
  type        = number
  default     = 10

  validation {
    condition     = var.sa_key_ttl_days == null ? true : var.sa_key_ttl_rotation_buffer_days < var.sa_key_ttl_days
    error_message = "ttl_rotation_buffer_days must be less than sa_key_ttl_days"
  }
}


variable "role_assignment_role" {
  description = "The name of the role to assign (e.g. owner, editor, viewer)."
  type        = string
}
