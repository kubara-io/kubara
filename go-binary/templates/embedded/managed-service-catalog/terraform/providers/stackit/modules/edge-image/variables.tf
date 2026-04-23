variable "project_id" {
  type        = string
  description = "STACKIT project ID."
}

variable "name" {
  type        = string
  description = "Name of the uploaded Edge image."
}

variable "local_file_path" {
  type        = string
  description = "Local path to the image artifact that Terraform uploads."
}

variable "disk_format" {
  type        = string
  description = "Disk format of the image."
  default     = "raw"
}

variable "min_disk_size" {
  type        = number
  description = "Minimum disk size of the image."
  default     = 30

  validation {
    condition     = var.min_disk_size > 0
    error_message = "min_disk_size must be greater than zero."
  }
}

variable "operating_system" {
  type        = string
  description = "Operating system of the uploaded image."
  default     = "linux"
}

variable "operating_system_distro" {
  type        = string
  description = "Operating system distro of the uploaded image."
  default     = "talos"
}

variable "operating_system_version" {
  type        = string
  description = "Operating system version of the uploaded image."
  default     = "v1.9.5"
}
