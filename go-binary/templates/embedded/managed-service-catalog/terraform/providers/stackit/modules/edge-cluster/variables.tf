variable "name" {
  type        = string
  description = "Name of the edgecloud cluster."

}


variable "project_id" {
  type        = string
  description = "The StackIT ProjectID."
}

variable "labels" {
  type        = map(string)
  description = "Labels for the edgecloud cluster."
  default = {
    "role" = "controlplane"
  }
}

variable "edgecloud_image_id" {
  type        = string
  description = "Edgecloud image ID."
}


variable "controlplane" {
  description = "Config for control plane nodes"
  type = object({
    flavor                   = string
    volume_size              = number
    volume_performance_class = string
  })
}


variable "availability_zone" {
  type        = string
  description = "Availability zone of the image."
  default     = "eu01-1"

}


variable "ipv4_nameservers" {
  type        = list(string)
  description = "IPv4 nameservers for the edgecloud network."
  default     = ["1.1.1.1", "1.0.0.1", "8.8.8.8"]

}

variable "ipv4_prefix" {
  type        = string
  description = "IPv4 prefix for the edgecloud network."
}
