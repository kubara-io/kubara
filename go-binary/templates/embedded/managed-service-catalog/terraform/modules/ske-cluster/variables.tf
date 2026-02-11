variable "project_id" {
  description = "STACKIT project ID to associate the cluster with"
  type        = string
}

variable "name" {
  description = "Name of the SKE cluster. Can only container 11 characters"
  type        = string
}

variable "region" {
  description = "Region for the cluster; falls back to provider region if null"
  type        = string
  default     = "eu01"
}

variable "kubernetes_version_min" {
  description = "Minimum Kubernetes version (e.g. \"1.31.7\"); uses latest if unset"
  type        = string
  default     = "1.31.8"
}

variable "node_pools" {
  description = <<EOF
List of node pools. Each element must be an object with:
- name               = string
- machine_type       = string
- os_version_min     = string
- minimum            = number
- maximum            = number
- availability_zones = list(string)
- labels             = map(string)
- taints             = list(object({
    key    = string
    value  = string
    effect = string
  }))
EOF
  type = list(object({
    name               = string
    machine_type       = string
    os_version_min     = string
    minimum            = number
    maximum            = number
    availability_zones = list(string)
    labels             = map(string)
    taints = list(object({
      key    = string
      value  = string
      effect = string
    }))
  }))
  default = []
}

variable "ske_maintenance" {
  type = object({
    enable_kubernetes_version_updates    = bool
    enable_machine_image_version_updates = bool
    start                                = string
    end                                  = string
  })

  default = {
    enable_kubernetes_version_updates    = true
    enable_machine_image_version_updates = true
    start                                = "01:00:00Z"
    end                                  = "02:00:00Z"
  }
}


### Kubeconfig variables
variable "refresh" {
  description = "Refresh token for the SKE cluster"
  type        = bool
  default     = true

}

variable "create_kubeconfig_local" {
  type        = bool
  default     = false
  description = "If true, write the kubeconfig to a local file"
}

variable "kubeconfig_path" {
  type        = string
  default     = "~/.kube/config"
  description = "Filesystem path where the kubeconfig will be written if create_kubeconfig_local is true"
}


variable "expiration" {
  description = "Expiration time for the kubeconfig in seconds"
  type        = number
  default     = 86400 # 24h
}
