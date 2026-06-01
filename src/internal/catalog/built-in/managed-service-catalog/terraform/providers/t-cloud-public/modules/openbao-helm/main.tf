locals {
  image_pull_secrets = [
    for name in var.image_pull_secrets : {
      name = name
    }
  ]

  raft_config = trimspace(<<-EOT
    ui = true

    listener "tcp" {
      tls_disable = 1
      address = "[::]:8200"
      cluster_address = "[::]:8201"
    }

    storage "raft" {
      path = "/openbao/data"
    }

    service_registration "kubernetes" {}

    ${trimspace(var.seal_config)}
  EOT
  )
}

resource "helm_release" "this" {
  name             = var.release_name
  repository       = var.repository
  chart            = var.chart
  version          = var.chart_version
  namespace        = var.namespace
  create_namespace = var.create_namespace
  wait             = true
  timeout          = var.helm_timeout

  values = [
    yamlencode({
      global = {
        imagePullSecrets = local.image_pull_secrets
      }
      injector = {
        enabled = var.injector_enabled
      }
      server = {
        image = {
          registry   = var.image_registry
          repository = var.image_repository
          tag        = var.image_tag
          pullPolicy = var.image_pull_policy
        }
        extraEnvironmentVars       = var.extra_environment_vars
        extraSecretEnvironmentVars = var.extra_secret_environment_vars
        dataStorage = {
          enabled      = true
          size         = var.data_storage_size
          storageClass = var.data_storage_class == "" ? null : var.data_storage_class
        }
        ha = {
          enabled  = true
          replicas = var.replicas
          raft = {
            enabled   = true
            setNodeId = true
            config    = local.raft_config
          }
        }
      }
    })
  ]
}
