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

variable "enable_load_balancer" {
  description = "Create an external load balancer with an associated EIP."
  type        = bool
  default     = true
}

variable "load_balancer_eip_type" {
  description = "EIP type for the external load balancer."
  type        = string
  default     = "5_bgp"
}

variable "load_balancer_eip_bandwidth_size" {
  description = "Load balancer EIP bandwidth in Mbit/s."
  type        = number
  default     = 300
}
