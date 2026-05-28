output "vpc_id" {
  description = "VPC ID."
  value       = opentelekomcloud_vpc_v1.this.id
}

output "subnet_id" {
  description = "Subnet ID used by CCE resources."
  value       = opentelekomcloud_vpc_subnet_v1.this.id
}

output "subnet_network_id" {
  description = "Subnet network ID used by load balancers."
  value       = opentelekomcloud_vpc_subnet_v1.this.subnet_id
}

output "nat_gateway_id" {
  description = "NAT gateway ID, if created."
  value       = try(opentelekomcloud_nat_gateway_v2.this[0].id, "")
}

output "load_balancer_id" {
  description = "Load balancer ID, if created."
  value       = try(opentelekomcloud_lb_loadbalancer_v2.this[0].id, "")
}

output "load_balancer_public_ip" {
  description = "Public IP address assigned to the optional load balancer."
  value       = try(opentelekomcloud_vpc_eip_v1.load_balancer[0].publicip[0].ip_address, "")
}
