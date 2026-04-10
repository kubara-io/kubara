resource "vault_kv_secret_v2" "cluster_secrets" {
  mount = module.secretsmanager.instance_id
  name  = "cluster_secrets"

  data_json = jsonencode({
    vault_instance = {
      api_url     = "https://prod.sm.eu01.stackit.cloud"
      instance_id = module.secretsmanager.instance_id
      password    = module.secretsmanager.users["vault-user-ro"].password
      username    = module.secretsmanager.users["vault-user-ro"].username
      user_id     = module.secretsmanager.users["vault-user-ro"].user_id
    }
  })
}


resource "vault_kv_secret_v2" "dns_zone_admin" {
  mount = module.secretsmanager.instance_id
  name  = "dns_zone_admin"
  data_json = jsonencode({
    sa_key_json = module.iam.service_account_key_json
    project_id  = var.project_id
    zone_id     = module.dns_zone.zone_id
    dns_name    = module.dns_zone.name
  })
}



# Secrets for Grafana admin credentials
resource "random_string" "grafana_admin_user" {
  count   = var.grafana_admin_user == "" ? 1 : 0
  length  = 8
  special = false
}

resource "random_password" "grafana_admin_password" {
  count  = var.grafana_admin_password == "" ? 1 : 0
  length = 32
}
resource "vault_kv_secret_v2" "grafana_admin_credentials" {
  mount = module.secretsmanager.instance_id #"<your_vault_instance_id>", in this example it comes from a TF module, but you can also just place your instance ID here
  name  = "grafana_credentials"
  data_json = jsonencode({
    admin-user     = local.grafana_admin_user
    admin-password = local.grafana_admin_password
  })
}
