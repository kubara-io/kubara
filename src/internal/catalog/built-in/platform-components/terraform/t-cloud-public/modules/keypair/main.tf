resource "tls_private_key" "this" {
  algorithm = var.algorithm
}

resource "opentelekomcloud_compute_keypair_v2" "this" {
  name       = var.name
  public_key = tls_private_key.this.public_key_openssh
}
