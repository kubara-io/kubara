variable "project_id" {
  type        = string
  description = "STACKIT project ID."
}

variable "create" {
  type        = bool
  description = "Whether Terraform should create a new Edge Cloud instance. Set to false to reuse an existing instance_id."
  default     = true
}

variable "instance_id" {
  type        = string
  description = "Existing Edge Cloud instance ID to reuse when create is false."
  default     = ""

  validation {
    condition     = var.create || trimspace(var.instance_id) != ""
    error_message = "instance_id must be set when create is false."
  }
}

variable "display_name" {
  type        = string
  description = "Display name of the Edge Cloud instance (4-8 chars, valid hostname label)."
  default     = ""

  validation {
    condition     = !var.create || trimspace(var.display_name) != ""
    error_message = "display_name must be set when create is true."
  }

  validation {
    condition     = var.display_name == "" || can(regex("^[a-z0-9][a-z0-9-]{2,6}[a-z0-9]$", var.display_name))
    error_message = "display_name must be empty or a valid hostname label with a length between 4 and 8 characters."
  }
}

variable "description" {
  type        = string
  description = "Description for the Edge Cloud instance."
  default     = ""
}

variable "plan_name" {
  type        = string
  description = "Name of the Edge Cloud plan to use when create is true."
  default     = "preview"
}

variable "region" {
  type        = string
  description = "Region used for Edge Cloud resources."
  default     = "eu01"
}
