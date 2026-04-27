variable "name" {
  type        = string
  description = "Name of the image."

}

variable "project_id" {
  type        = string
  description = "The STACKIT ProjectID."
}

variable "disk_format" {
  type        = string
  description = "Disk format of the image."
  default     = "raw"
}

variable "local_file_path" {
  type        = string
  description = "Local file path of the image."

}

variable "min_disk_size" {
  type        = number
  description = "Minimum disk size of the image."
  default     = 10

}

variable "config" {
  type = object({
    operating_system         = string
    operating_system_distro  = string
    operating_system_version = string
  })
  description = "Configuration of the image."
  default = {
    operating_system         = "linux"
    operating_system_distro  = "talos"
    operating_system_version = "v1.9.5"
  }

}
