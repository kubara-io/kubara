variable "name" {
  description = "Name prefix for the KMS key alias."
  type        = string
}

variable "description" {
  description = "KMS key description."
  type        = string
  default     = "kubara managed CCE node pool volume encryption key"
}

variable "pending_days" {
  description = "Pending deletion days for the KMS key."
  type        = number
  default     = 7
}

variable "is_enabled" {
  description = "Whether the KMS key is enabled."
  type        = bool
  default     = true
}
