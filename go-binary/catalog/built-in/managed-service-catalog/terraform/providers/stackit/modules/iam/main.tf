resource "stackit_service_account" "this" {
  project_id = var.project_id
  name       = var.name
}

# Only created when sa_key_ttl_days is set (i.e. TTL-enabled mode)
resource "time_rotating" "rotate" {
  count         = var.sa_key_ttl_days != null ? 1 : 0
  rotation_days = var.sa_key_ttl_days - var.sa_key_ttl_rotation_buffer_days
}

# Service account key WITH TTL + automatic rotation
resource "stackit_service_account_key" "with_ttl" {
  count                 = var.sa_key_ttl_days != null ? 1 : 0
  project_id            = var.project_id
  service_account_email = stackit_service_account.this.email
  ttl_days              = var.sa_key_ttl_days

  rotate_when_changed = {
    rotation = time_rotating.rotate[0].id
  }
}

# Service account key WITHOUT TTL (default behavior)
resource "stackit_service_account_key" "no_ttl" {
  count                 = var.sa_key_ttl_days == null ? 1 : 0
  project_id            = var.project_id
  service_account_email = stackit_service_account.this.email
}


resource "stackit_authorization_project_role_assignment" "this" {

  resource_id = var.project_id
  role        = var.role_assignment_role
  subject     = stackit_service_account.this.email
}
