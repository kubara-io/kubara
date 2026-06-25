output "id" {
  description = "DNS zone ID."
  value       = opentelekomcloud_dns_zone_v2.this.id
}

output "name" {
  description = "DNS zone name."
  value       = opentelekomcloud_dns_zone_v2.this.name
}

output "type" {
  description = "DNS zone type."
  value       = opentelekomcloud_dns_zone_v2.this.type
}

output "ttl" {
  description = "DNS zone TTL."
  value       = opentelekomcloud_dns_zone_v2.this.ttl
}

output "masters" {
  description = "DNS zone master name servers."
  value       = opentelekomcloud_dns_zone_v2.this.masters
}
