output "ids" {
  description = "IAM agency IDs keyed by logical name."
  value = {
    for key, agency in opentelekomcloud_identity_agency_v3.this : key => agency.id
  }
}

output "names" {
  description = "IAM agency names keyed by logical name."
  value = {
    for key, agency in opentelekomcloud_identity_agency_v3.this : key => agency.name
  }
}
