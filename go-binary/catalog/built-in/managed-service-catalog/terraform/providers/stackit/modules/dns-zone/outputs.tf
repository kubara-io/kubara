output "id" {
  description = "Internal Terraform resource ID (project_id,zone_id)"
  value       = stackit_dns_zone.this.id
}

output "zone_id" {
  description = "Unique ID of the created DNS zone"
  value       = stackit_dns_zone.this.zone_id
}

output "primary_name_server" {
  description = "FQDN of the primary name server"
  value       = stackit_dns_zone.this.primary_name_server
}

output "record_count" {
  description = "Number of records in the zone"
  value       = stackit_dns_zone.this.record_count
}

output "serial_number" {
  description = "Serial number of the zone"
  value       = stackit_dns_zone.this.serial_number
}

output "state" {
  description = "Current state (e.g. CREATE_SUCCEEDED)"
  value       = stackit_dns_zone.this.state
}

output "visibility" {
  description = "Zone visibility (e.g. public)"
  value       = stackit_dns_zone.this.visibility
}

output "name" {
  description = "Name of the DNS zone"
  value       = stackit_dns_zone.this.name
}
