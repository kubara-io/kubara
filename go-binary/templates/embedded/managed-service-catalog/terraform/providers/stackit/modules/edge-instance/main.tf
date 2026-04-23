data "stackit_edgecloud_plans" "this" {
  count      = var.create ? 1 : 0
  project_id = var.project_id
}

locals {
  available_plan_names = var.create ? sort([
    for plan in data.stackit_edgecloud_plans.this[0].plans : plan.name
  ]) : []

  matching_plans = var.create ? [
    for plan in data.stackit_edgecloud_plans.this[0].plans : plan
    if lower(plan.name) == lower(var.plan_name)
  ] : []
}

resource "stackit_edgecloud_instance" "this" {
  count = var.create ? 1 : 0

  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
  description  = var.description == "" ? null : var.description
  plan_id      = one(local.matching_plans).id

  lifecycle {
    precondition {
      condition = length(local.matching_plans) == 1
      error_message = format(
        "Expected exactly one Edge Cloud plan match for '%s' but found %d. Available plans: %s",
        var.plan_name,
        length(local.matching_plans),
        join(", ", local.available_plan_names),
      )
    }
  }
}
