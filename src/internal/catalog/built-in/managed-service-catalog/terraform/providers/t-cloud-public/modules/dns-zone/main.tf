locals {
  zone_name = endswith(var.name, ".") ? var.name : "${var.name}."
}

resource "opentelekomcloud_dns_zone_v2" "this" {
  name        = local.zone_name
  email       = var.email
  type        = var.type
  ttl         = var.ttl
  description = var.description
  tags        = var.tags

  dynamic "router" {
    for_each = var.routers
    content {
      router_id     = router.value.router_id
      router_region = router.value.router_region
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}
