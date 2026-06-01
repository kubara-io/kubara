locals {
  image_pull_secrets = [
    for name in var.image_pull_secrets : {
      name = name
    }
  ]

  ingress_path = var.ingress_path == "/" ? "/" : trimsuffix(var.ingress_path, "/")
  ingress_url  = var.ingress_enabled && var.ingress_host != "" ? "https://${var.ingress_host}${local.ingress_path == "/" ? "" : local.ingress_path}" : null

  ingress_default_annotations = local.ingress_path == "/" ? {} : {
    "traefik.ingress.kubernetes.io/app-root"           = "${local.ingress_path}/ui/"
    "traefik.ingress.kubernetes.io/router.middlewares" = "${var.namespace}-openbao-redirect-noslash@kubernetescrd,${var.namespace}-openbao-strip-prefix@kubernetescrd"
  }

  ingress_tls = var.ingress_tls_secret_name == "" ? [] : [
    {
      secretName = var.ingress_tls_secret_name
      hosts      = [var.ingress_host]
    }
  ]

  ingress_extra_objects = var.ingress_enabled && local.ingress_path != "/" ? [
    yamlencode({
      apiVersion = "traefik.io/v1alpha1"
      kind       = "Middleware"
      metadata = {
        name      = "openbao-redirect-noslash"
        namespace = var.namespace
      }
      spec = {
        redirectRegex = {
          permanent   = true
          regex       = "^(https?://[^/]+${local.ingress_path})$"
          replacement = "$${1}/"
        }
      }
    }),
    yamlencode({
      apiVersion = "traefik.io/v1alpha1"
      kind       = "Middleware"
      metadata = {
        name      = "openbao-strip-prefix"
        namespace = var.namespace
      }
      spec = {
        replacePathRegex = {
          regex       = "^${local.ingress_path}/(.*)"
          replacement = "/$${1}"
        }
      }
    }),
  ] : []

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
  create_namespace = true
  wait             = false

  values = [
    yamlencode({
      global = {
        imagePullSecrets = local.image_pull_secrets
      }
      injector = {
        enabled = var.injector_enabled
      }
      server = {
        extraEnvironmentVars = var.extra_environment_vars
        dataStorage = {
          enabled      = true
          size         = var.data_storage_size
          storageClass = var.data_storage_class == "" ? null : var.data_storage_class
        }
        ha = {
          enabled  = true
          replicas = var.replicas
          apiAddr  = local.ingress_url
          raft = {
            enabled   = true
            setNodeId = true
            config    = local.raft_config
          }
        }
        ingress = {
          enabled          = var.ingress_enabled
          ingressClassName = var.ingress_class_name == "" ? null : var.ingress_class_name
          pathType         = "Prefix"
          activeService    = true
          annotations      = merge(local.ingress_default_annotations, var.ingress_annotations)
          hosts = [
            {
              host  = var.ingress_host
              paths = [local.ingress_path]
            }
          ]
          tls = local.ingress_tls
        }
      }
      extraObjects = local.ingress_extra_objects
    })
  ]
}
