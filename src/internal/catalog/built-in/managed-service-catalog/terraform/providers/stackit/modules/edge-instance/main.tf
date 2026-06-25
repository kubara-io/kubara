data "stackit_edgecloud_plans" "this" {
  project_id = var.project_id
}

resource "stackit_edgecloud_instance" "this" {
  project_id   = var.project_id
  region       = var.region
  display_name = var.display_name
  # Send null instead of an empty string so the provider treats it as unset.
  description = var.description == "" ? null : var.description
  # STEC is beta and currently exposes one available plan for the project.
  # one() does not choose arbitrarily; it fails if the API returns zero or multiple plans.
  plan_id = one(data.stackit_edgecloud_plans.this.plans).id
}

resource "stackit_edgecloud_kubeconfig" "this" {
  project_id  = var.project_id
  region      = var.region
  instance_id = stackit_edgecloud_instance.this.instance_id
  expiration  = var.expiration
}
