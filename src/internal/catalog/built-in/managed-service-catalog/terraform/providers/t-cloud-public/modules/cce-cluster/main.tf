resource "opentelekomcloud_vpc_eip_v1" "public_endpoint" {
  count = var.enable_public_endpoint ? 1 : 0

  publicip {
    type = var.public_endpoint_eip_type
  }

  bandwidth {
    name        = var.public_endpoint_eip_bandwidth_name != "" ? var.public_endpoint_eip_bandwidth_name : "eip-cce-${var.name}"
    size        = var.public_endpoint_eip_bandwidth_size
    share_type  = var.public_endpoint_eip_bandwidth_share_type
    charge_mode = var.public_endpoint_eip_bandwidth_charge_mode
  }
}

resource "opentelekomcloud_cce_cluster_v3" "this" {
  name                   = var.name
  cluster_type           = var.cluster_type
  flavor_id              = var.cluster_flavor_id
  cluster_version        = var.kubernetes_version_min
  vpc_id                 = var.vpc_id
  subnet_id              = var.subnet_id
  container_network_type = var.container_network_type
  billing_mode           = 0
  description            = var.description

  # Binds the EIP to the CCE master so the API server is reachable from outside
  # the VPC. The provider takes the public IP as a string, not an EIP ID.
  eip = var.enable_public_endpoint ? opentelekomcloud_vpc_eip_v1.public_endpoint[0].publicip[0].ip_address : null

  timeouts {
    create = "60m"
    delete = "60m"
  }

  lifecycle {
    ignore_changes = [
      name,
    ]
  }
}

resource "opentelekomcloud_cce_node_pool_v3" "this" {
  for_each = {
    for node_pool in var.node_pools : node_pool.name => node_pool
  }

  cluster_id         = opentelekomcloud_cce_cluster_v3.this.id
  name               = each.value.name
  flavor             = each.value.flavor
  initial_node_count = each.value.initial_node_count
  availability_zone  = each.value.availability_zone
  key_pair           = var.key_pair_name
  runtime            = each.value.runtime
  os                 = each.value.os
  scale_enable       = each.value.scale_enable
  docker_base_size   = each.value.docker_base_size

  root_volume {
    size       = each.value.root_volume.size
    volumetype = each.value.root_volume.volumetype
    kms_id     = var.node_storage_kms_id
  }

  dynamic "data_volumes" {
    for_each = each.value.data_volumes
    content {
      size       = data_volumes.value.size
      volumetype = data_volumes.value.volumetype
      kms_id     = var.node_storage_kms_id
    }
  }

  lifecycle {
    ignore_changes = [
      initial_node_count,
    ]
  }
}

data "opentelekomcloud_identity_project_v3" "current" {}

locals {
  addon_image_endpoint = data.opentelekomcloud_identity_project_v3.current.region == "eu-de" ? "100.125.7.25:20202" : "swr.eu-nl.otc.t-systems.com"
  enabled_addons = {
    for name, addon in var.addons : name => addon
    if addon.enabled
  }
}

resource "opentelekomcloud_cce_addon_v3" "this" {
  for_each = local.enabled_addons

  template_name    = each.key
  template_version = each.value.version
  cluster_id       = opentelekomcloud_cce_cluster_v3.this.id

  values {
    basic = merge({
      swr_addr = local.addon_image_endpoint
      swr_user = "cce-addons"
    }, each.value.basic)
    custom = each.value.custom
  }

  depends_on = [opentelekomcloud_cce_node_pool_v3.this]
}

data "opentelekomcloud_cce_cluster_kubeconfig_v3" "this" {
  cluster_id = opentelekomcloud_cce_cluster_v3.this.id
  duration   = var.kubeconfig_duration

  depends_on = [opentelekomcloud_cce_addon_v3.this]
}

resource "local_file" "kubeconfig" {
  count           = var.create_kubeconfig_local ? 1 : 0
  content         = data.opentelekomcloud_cce_cluster_kubeconfig_v3.this.kubeconfig
  filename        = var.kubeconfig_path
  file_permission = "0600"
}
