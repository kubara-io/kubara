resource "kubernetes_storage_class_v1" "this" {
  for_each = var.storage_classes

  metadata {
    name        = each.key
    annotations = each.value.annotations
    labels      = each.value.labels
  }

  storage_provisioner    = each.value.storage_provisioner
  reclaim_policy         = each.value.reclaim_policy
  volume_binding_mode    = each.value.volume_binding_mode
  allow_volume_expansion = each.value.allow_volume_expansion
  parameters             = each.value.parameters
}
