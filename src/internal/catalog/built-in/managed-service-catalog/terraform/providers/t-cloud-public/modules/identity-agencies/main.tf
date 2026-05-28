resource "opentelekomcloud_identity_agency_v3" "this" {
  for_each = var.agencies

  name                  = each.value.name
  description           = each.value.description
  delegated_domain_name = each.value.delegated_domain_name
  domain_roles          = each.value.domain_roles

  dynamic "project_role" {
    for_each = each.value.project_roles
    content {
      project      = project_role.value.project
      all_projects = project_role.value.all_projects
      roles        = project_role.value.roles
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}
