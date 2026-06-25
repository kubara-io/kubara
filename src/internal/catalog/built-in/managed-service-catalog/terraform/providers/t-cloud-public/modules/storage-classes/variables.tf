variable "storage_classes" {
  description = "Kubernetes StorageClasses keyed by StorageClass name."
  type = map(object({
    annotations            = optional(map(string), {})
    labels                 = optional(map(string), {})
    parameters             = map(string)
    storage_provisioner    = optional(string, "everest-csi-provisioner")
    reclaim_policy         = optional(string, "Retain")
    volume_binding_mode    = optional(string, "WaitForFirstConsumer")
    allow_volume_expansion = optional(bool, true)
  }))
  default = {}
}
