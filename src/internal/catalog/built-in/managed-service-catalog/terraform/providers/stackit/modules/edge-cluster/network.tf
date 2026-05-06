resource "stackit_public_ip" "public_ip" {
  project_id           = var.project_id
  network_interface_id = stackit_network_interface.this.network_interface_id
  labels               = var.labels

}
resource "stackit_security_group" "public_ip_sec_group" {
  project_id = var.project_id
  name       = var.name
}

resource "stackit_security_group_rule" "public_ip_sec_group_ingress_80" {
  project_id        = var.project_id
  security_group_id = stackit_security_group.public_ip_sec_group.security_group_id
  direction         = "ingress"
  description       = "allow ingress port 80"
  protocol = {
    name = "tcp"
  }
  port_range = {
    min = 80
    max = 80
  }
}

resource "stackit_security_group_rule" "public_ip_sec_group_ingress_443" {
  project_id        = var.project_id
  security_group_id = stackit_security_group.public_ip_sec_group.security_group_id
  direction         = "ingress"
  description       = "allow ingress port 443"
  protocol = {
    name = "tcp"
  }
  port_range = {
    min = 443
    max = 443
  }
}


resource "stackit_network" "edgecloud-network" {
  project_id         = var.project_id
  name               = var.name
  ipv4_nameservers   = var.ipv4_nameservers
  ipv4_prefix        = var.ipv4_prefix
  ipv4_prefix_length = 24
}
