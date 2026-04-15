resource "stackit_volume" "this" {
  project_id        = var.project_id
  name              = var.name
  availability_zone = var.availability_zone
  size              = var.controlplane.volume_size
  performance_class = var.controlplane.volume_performance_class
  source = {
    type = "image"
    id   = var.edgecloud_image_id
  }
}

resource "stackit_network_interface" "this" {
  project_id         = var.project_id
  network_id         = stackit_network.edgecloud-network.network_id
  name               = var.name
  security_group_ids = [stackit_security_group.public_ip_sec_group.security_group_id]
}

resource "stackit_server" "this" {
  project_id   = var.project_id
  name         = var.name
  machine_type = var.controlplane.flavor

  boot_volume = {
    source_type = "volume"
    source_id   = stackit_volume.this.volume_id
  }
  network_interfaces = [stackit_network_interface.this.network_interface_id]
}
