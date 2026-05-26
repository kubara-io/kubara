variable "name" {
  type        = string
  description = "Name prefix used for generated host resources."
}

variable "project_id" {
  type        = string
  description = "STACKIT project ID."
}

variable "image_id" {
  type        = string
  description = "Image ID used to create boot volumes for all hosts."

  validation {
    condition     = trimspace(var.image_id) != ""
    error_message = "image_id must be set and must not be empty."
  }
}

variable "network_name" {
  type        = string
  description = "Name of the shared network created for all edge hosts."
}

variable "security_group_name" {
  type        = string
  description = "Name of the shared security group attached to all host interfaces."
}

variable "ipv4_prefix" {
  type        = string
  description = "IPv4 network prefix (without mask) used for the host network."
}

variable "ipv4_prefix_length" {
  type        = number
  description = "IPv4 prefix length used for the host network."
  default     = 24

  validation {
    condition     = var.ipv4_prefix_length >= 8 && var.ipv4_prefix_length <= 30
    error_message = "ipv4_prefix_length must be between 8 and 30."
  }
}

variable "ipv4_nameservers" {
  type        = list(string)
  description = "List of nameservers for the host network."
  default     = ["1.1.1.1", "1.0.0.1", "8.8.8.8"]
}

variable "ingress_tcp_ports" {
  type        = list(number)
  description = "List of TCP ports opened on the shared security group."
  default     = [80, 443]
}

variable "common_labels" {
  type        = map(string)
  description = "Labels added to public IP resources for all nodes."
  default     = {}
}

variable "nodes" {
  description = "Edge hosts to provision."
  type = list(object({
    name                     = string
    role                     = string
    flavor                   = string
    volume_size              = number
    volume_performance_class = string
    availability_zone        = string
    assign_public_ip         = optional(bool, true)
    labels                   = optional(map(string), {})
  }))

  validation {
    condition     = length(var.nodes) > 0
    error_message = "nodes must contain at least one host definition."
  }

  validation {
    condition     = length(var.nodes) == length(toset([for node in var.nodes : node.name]))
    error_message = "Each node name must be unique."
  }

  validation {
    condition     = alltrue([for node in var.nodes : contains(["controlplane", "worker"], lower(node.role))])
    error_message = "Each node role must be either controlplane or worker."
  }
}
