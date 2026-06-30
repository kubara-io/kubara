resource "random_id" "suffix" {
  byte_length = 4
}

resource "opentelekomcloud_kms_key_v1" "this" {
  key_alias       = "${var.name}-${random_id.suffix.hex}"
  key_description = var.description
  pending_days    = var.pending_days
  is_enabled      = tostring(var.is_enabled)
}
