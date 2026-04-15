resource "stackit_dns_zone" "this" {
  project_id = var.project_id
  name       = var.name
  dns_name   = var.dns_name

  contact_email   = var.contact_email
  acl             = var.acl
  default_ttl     = var.default_ttl
  description     = var.description
  type            = var.type
  is_reverse_zone = var.is_reverse_zone
  active          = var.active
  negative_cache  = var.negative_cache
  refresh_time    = var.refresh_time
  retry_time      = var.retry_time
}
