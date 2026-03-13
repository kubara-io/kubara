# 1) Create the Secrets Manager instance
resource "stackit_secretsmanager_instance" "this" {
  project_id = var.project_id
  name       = var.name
  acls       = var.acls
}

# 2) Iterate over the user list to create each user
resource "stackit_secretsmanager_user" "this" {
  for_each    = { for u in var.users : u.description => u }
  project_id  = var.project_id
  instance_id = stackit_secretsmanager_instance.this.instance_id

  description   = each.value.description
  write_enabled = each.value.write_enabled

  # ensure instance exists before creating users
  depends_on = [stackit_secretsmanager_instance.this]
}
