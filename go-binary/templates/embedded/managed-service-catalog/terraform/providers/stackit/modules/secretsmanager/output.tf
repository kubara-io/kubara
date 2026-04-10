# Instance outputs
output "instance_id" {
  description = "ID of the created Secrets Manager instance"
  value       = stackit_secretsmanager_instance.this.instance_id
}

output "instance" {
  description = "Map of instance attributes"
  value = {
    id          = stackit_secretsmanager_instance.this.id
    instance_id = stackit_secretsmanager_instance.this.instance_id
    name        = var.name
    acls        = var.acls
  }
}


# Users outputs
output "users" {
  description = "Map of created users, keyed by description"
  value = {
    for key, user in stackit_secretsmanager_user.this :
    key => {
      user_id       = user.user_id
      username      = user.username
      password      = user.password
      write_enabled = user.write_enabled
    }
  }
  sensitive = true
}
