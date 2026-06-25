variable "name" {
  description = "Name prefix for network resources."
  type        = string
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC."
  type        = string
  default     = "10.0.0.0/16"
}

variable "subnet_cidr" {
  description = "CIDR block for the subnet."
  type        = string
  default     = "10.0.1.0/24"
}

variable "subnet_gateway_ip" {
  description = "Gateway IP for the subnet."
  type        = string
  default     = "10.0.1.1"
}

variable "subnet_dns_list" {
  description = "DNS servers for the subnet."
  type        = list(string)
  default     = ["100.125.4.25", "100.125.129.199"]
}

variable "enable_nat_gateway" {
  description = "Create a NAT gateway and SNAT rule for subnet egress."
  type        = bool
  default     = true
}

variable "nat_gateway_spec" {
  description = "NAT gateway spec."
  type        = string
  default     = "0"
}

variable "nat_eip_type" {
  description = "EIP type for the NAT gateway."
  type        = string
  default     = "5_mailbgp"
}

variable "nat_eip_bandwidth_size" {
  description = "NAT gateway EIP bandwidth in Mbit/s."
  type        = number
  default     = 300
}

variable "enable_shared_load_balancer" {
  description = "Create the shared external load balancer used by cluster ingress, with an associated EIP."
  type        = bool
  default     = true
}

variable "enable_dedicated_load_balancer" {
  description = "Create an additional dedicated external load balancer with an associated EIP."
  type        = bool
  default     = false
}

variable "load_balancer_eip_type" {
  description = "EIP type for the shared external load balancer."
  type        = string
  default     = "5_bgp"
}

variable "load_balancer_eip_bandwidth_size" {
  description = "Shared load balancer EIP bandwidth in Mbit/s."
  type        = number
  default     = 300
}

variable "dedicated_load_balancer_eip_type" {
  description = "EIP type for the dedicated external load balancer."
  type        = string
  default     = "5_bgp"
}

variable "dedicated_load_balancer_eip_bandwidth_size" {
  description = "Dedicated load balancer EIP bandwidth in Mbit/s."
  type        = number
  default     = 300
}

variable "dedicated_load_balancer_availability_zones" {
  description = "Availability zones for the dedicated external load balancer."
  type        = list(string)
  default     = ["eu-de-01"]

  validation {
    condition     = length(var.dedicated_load_balancer_availability_zones) > 0
    error_message = "dedicated_load_balancer_availability_zones must contain at least one availability zone."
  }
}

variable "dedicated_load_balancer_l4_flavor_name" {
  description = "Layer-4 flavor name for the dedicated external load balancer. Set to an empty string to skip L4."
  type        = string
  default     = "L4_flavor.elb.s1.small"
}

variable "dedicated_load_balancer_l7_flavor_name" {
  description = "Layer-7 flavor name for the dedicated external load balancer. Set to an empty string to skip L7."
  type        = string
  default     = "L7_flavor.elb.s1.small"
}
