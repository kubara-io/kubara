locals {
  nodes_by_name = {
    for node in var.nodes : node.name => node
  }

  nodes_with_public_ip = {
    for name, node in local.nodes_by_name : name => node
    if node.assign_public_ip
  }
}

resource "stackit_network" "this" {
  project_id         = var.project_id
  name               = var.network_name
  ipv4_nameservers   = var.ipv4_nameservers
  ipv4_prefix        = var.ipv4_prefix
  ipv4_prefix_length = var.ipv4_prefix_length
}

resource "stackit_security_group" "this" {
  project_id = var.project_id
  name       = var.security_group_name
}

resource "stackit_security_group_rule" "ingress_tcp" {
  for_each = toset([for p in var.ingress_tcp_ports : tostring(p)])

  project_id        = var.project_id
  security_group_id = stackit_security_group.this.security_group_id
  direction         = "ingress"
  description       = "allow ingress tcp ${each.value}"

  protocol = {
    name = "tcp"
  }

  port_range = {
    min = tonumber(each.value)
    max = tonumber(each.value)
  }
}

resource "stackit_volume" "this" {
  for_each = local.nodes_by_name

  project_id        = var.project_id
  name              = "${var.name}-${each.key}-volume"
  availability_zone = each.value.availability_zone
  size              = each.value.volume_size
  performance_class = each.value.volume_performance_class

  source = {
    type = "image"
    id   = var.image_id
  }
}

resource "stackit_network_interface" "this" {
  for_each = local.nodes_by_name

  project_id = var.project_id
  network_id = stackit_network.this.network_id
  name       = "${var.name}-${each.key}-nic"

  security_group_ids = [stackit_security_group.this.security_group_id]
}

resource "stackit_server" "this" {
  for_each = local.nodes_by_name

  project_id   = var.project_id
  name         = each.value.name
  machine_type = each.value.flavor

  boot_volume = {
    source_type = "volume"
    source_id   = stackit_volume.this[each.key].volume_id
  }

  network_interfaces = [stackit_network_interface.this[each.key].network_interface_id]
}

resource "stackit_public_ip" "this" {
  for_each = local.nodes_with_public_ip

  project_id           = var.project_id
  network_interface_id = stackit_network_interface.this[each.key].network_interface_id
  labels               = merge(var.common_labels, each.value.labels, { role = lower(each.value.role) })
}
