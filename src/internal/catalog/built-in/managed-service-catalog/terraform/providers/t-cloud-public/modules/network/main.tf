resource "opentelekomcloud_vpc_v1" "this" {
  name = "${var.name}-vpc"
  cidr = var.vpc_cidr
}

resource "opentelekomcloud_vpc_subnet_v1" "this" {
  name       = "${var.name}-subnet"
  cidr       = var.subnet_cidr
  vpc_id     = opentelekomcloud_vpc_v1.this.id
  gateway_ip = var.subnet_gateway_ip
  dns_list   = var.subnet_dns_list
}

resource "opentelekomcloud_nat_gateway_v2" "this" {
  count = var.enable_nat_gateway ? 1 : 0

  name                = "nat-gw-${var.name}"
  description         = "NAT gateway created by kubara"
  spec                = var.nat_gateway_spec
  router_id           = opentelekomcloud_vpc_v1.this.id
  internal_network_id = opentelekomcloud_vpc_subnet_v1.this.id
}

resource "opentelekomcloud_vpc_eip_v1" "nat" {
  count = var.enable_nat_gateway ? 1 : 0

  publicip {
    type = var.nat_eip_type
  }
  bandwidth {
    name        = "eip-nat-gw-${var.name}"
    size        = var.nat_eip_bandwidth_size
    share_type  = "PER"
    charge_mode = "traffic"
  }
}

resource "opentelekomcloud_nat_snat_rule_v2" "this" {
  count = var.enable_nat_gateway ? 1 : 0

  nat_gateway_id = opentelekomcloud_nat_gateway_v2.this[0].id
  floating_ip_id = opentelekomcloud_vpc_eip_v1.nat[0].id
  cidr           = var.subnet_cidr
  source_type    = 0
}

locals {
  create_shared_load_balancer    = var.enable_load_balancer && var.load_balancer_type == "shared"
  create_dedicated_load_balancer = var.enable_load_balancer && var.load_balancer_type == "dedicated"
}

data "opentelekomcloud_lb_flavor_v3" "dedicated_l4" {
  count = local.create_dedicated_load_balancer && var.dedicated_load_balancer_l4_flavor_name != "" ? 1 : 0

  name = var.dedicated_load_balancer_l4_flavor_name
}

data "opentelekomcloud_lb_flavor_v3" "dedicated_l7" {
  count = local.create_dedicated_load_balancer && var.dedicated_load_balancer_l7_flavor_name != "" ? 1 : 0

  name = var.dedicated_load_balancer_l7_flavor_name
}

resource "opentelekomcloud_lb_loadbalancer_v2" "this" {
  count = local.create_shared_load_balancer ? 1 : 0

  name          = "elb-${var.name}"
  vip_subnet_id = opentelekomcloud_vpc_subnet_v1.this.subnet_id
}

resource "opentelekomcloud_vpc_eip_v1" "load_balancer" {
  count = local.create_shared_load_balancer ? 1 : 0

  publicip {
    type = var.load_balancer_eip_type
  }
  bandwidth {
    name        = "eip-elb-${var.name}"
    size        = var.load_balancer_eip_bandwidth_size
    share_type  = "PER"
    charge_mode = "traffic"
  }
}

resource "opentelekomcloud_networking_floatingip_associate_v2" "load_balancer" {
  count = local.create_shared_load_balancer ? 1 : 0

  floating_ip = opentelekomcloud_vpc_eip_v1.load_balancer[0].publicip[0].ip_address
  port_id     = opentelekomcloud_lb_loadbalancer_v2.this[0].vip_port_id
}

resource "opentelekomcloud_vpc_eip_v1" "dedicated_load_balancer" {
  count = local.create_dedicated_load_balancer ? 1 : 0

  publicip {
    type = var.load_balancer_eip_type
  }
  bandwidth {
    name        = "eip-elb-${var.name}"
    size        = var.load_balancer_eip_bandwidth_size
    share_type  = "PER"
    charge_mode = "traffic"
  }
}

resource "opentelekomcloud_lb_loadbalancer_v3" "dedicated" {
  count = local.create_dedicated_load_balancer ? 1 : 0

  name               = "elb-${var.name}"
  router_id          = opentelekomcloud_vpc_v1.this.id
  network_ids        = [opentelekomcloud_vpc_subnet_v1.this.subnet_id]
  availability_zones = var.dedicated_load_balancer_availability_zones
  l4_flavor          = var.dedicated_load_balancer_l4_flavor_name != "" ? data.opentelekomcloud_lb_flavor_v3.dedicated_l4[0].id : null
  l7_flavor          = var.dedicated_load_balancer_l7_flavor_name != "" ? data.opentelekomcloud_lb_flavor_v3.dedicated_l7[0].id : null

  public_ip {
    id = opentelekomcloud_vpc_eip_v1.dedicated_load_balancer[0].id
  }
}
