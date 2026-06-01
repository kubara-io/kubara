locals {
  image_pull_secrets = [
    for name in var.image_pull_secrets : {
      name = name
    }
  ]

  ingress_path = var.ingress_path == "/" ? "/" : trimsuffix(var.ingress_path, "/")
  ingress_url  = var.ingress_enabled && var.ingress_host != "" ? "https://${var.ingress_host}${local.ingress_path == "/" ? "" : local.ingress_path}" : null

  # The traefik.io/v1alpha1 Middleware CRD is only available after Traefik is
  # installed in the cluster. When OpenBao is rolled out before Traefik (the
  # default kubara flow), creating the middlewares here breaks the Helm
  # release. Set ingress_create_traefik_middlewares = true once Traefik and
  # its CRDs exist (typically on a follow-up apply).
  ingress_create_middlewares  = var.ingress_enabled && local.ingress_path != "/" && var.ingress_create_traefik_middlewares
  ingress_default_annotations = local.ingress_create_middlewares ? {
    "traefik.ingress.kubernetes.io/app-root"           = "${local.ingress_path}/ui/"
    "traefik.ingress.kubernetes.io/router.middlewares" = "${var.namespace}-openbao-redirect-noslash@kubernetescrd,${var.namespace}-openbao-strip-prefix@kubernetescrd"
  } : {}

  ingress_tls = var.ingress_tls_secret_name == "" ? [] : [
    {
      secretName = var.ingress_tls_secret_name
      hosts      = [var.ingress_host]
    }
  ]

  ingress_extra_objects = local.ingress_create_middlewares ? [
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

  # HA-Raft needs each pod to discover its peers. Without retry_join, only
  # the pod where you run `bao operator init` ever joins the cluster; the
  # other pods stay uninitialized and unseal calls against them fail. The
  # OpenBao headless service is named "<release>-internal" by the chart and
  # exposes per-pod DNS entries like openbao-0.openbao-internal.
  raft_retry_joins = join("\n      ", [
    for i in range(var.replicas) :
    "retry_join {\n        leader_api_addr = \"http://${var.release_name}-${i}.${var.release_name}-internal:8200\"\n      }"
  ])

  raft_config = trimspace(<<-EOT
    ui = true
    disabled_mlock = true

    listener "tcp" {
      tls_disable = 1
      address = "[::]:8200"
      cluster_address = "[::]:8201"
    }

    storage "raft" {
      path = "/openbao/data"

      ${local.raft_retry_joins}
    }

    service_registration "kubernetes" {}

    ${trimspace(var.seal_config)}
  EOT
  )

  # Tell each pod its own externally-reachable API address, used by Raft for
  # standby-to-leader request forwarding. Kubernetes substitutes $(HOSTNAME)
  # at pod start, so this resolves to per-pod URLs like
  # http://openbao-0.openbao-internal:8200.
  per_pod_api_addr_env = {
    BAO_API_ADDR = "http://$(HOSTNAME).${var.release_name}-internal:8200"
  }
  merged_extra_env = merge(local.per_pod_api_addr_env, var.extra_environment_vars)

  # The OpenBao Helm chart applies its affinity field through `tpl`, so it
  # must be provided as a YAML string, not a structured object. The default
  # chart value uses required anti-affinity on hostname, which leaves the
  # third pod unscheduled when the cluster only has two worker nodes (a
  # common kubara CCE setup). preferredDuringScheduling spreads pods across
  # nodes when possible but allows co-location on the same node if no other
  # node is available.
  server_affinity = <<-EOT
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            labelSelector:
              matchLabels:
                app.kubernetes.io/name: openbao
                component: server
            topologyKey: kubernetes.io/hostname
  EOT
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
        extraEnvironmentVars = local.merged_extra_env
        updateStrategyType   = "RollingUpdate"
        affinity             = local.server_affinity
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
