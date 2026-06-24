variable "project_id" {
  type        = string
  description = "STACKIT project ID."
}

variable "display_name" {
  type        = string
  description = "Display name of the Edge Cloud instance. Can only contain 8 characters."

  validation {
    condition     = trimspace(var.display_name) != ""
    error_message = "display_name must be set."
  }
}

variable "description" {
  type        = string
  description = "Description for the Edge Cloud instance."
  default     = ""
}

variable "region" {
  type        = string
  description = "Region used for Edge Cloud resources."
  default     = "eu01"
}

variable "expiration" {
  type        = number
  description = "Expiration time for the kubeconfig in seconds."
  default     = 86400 # 24h
}
