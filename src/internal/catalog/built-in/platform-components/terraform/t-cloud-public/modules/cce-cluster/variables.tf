variable "name" {
  description = "Name of the CCE cluster."
  type        = string
}

variable "vpc_id" {
  description = "VPC ID for the CCE cluster."
  type        = string
}

variable "subnet_id" {
  description = "Subnet ID for the CCE cluster."
  type        = string
}

variable "key_pair_name" {
  description = "Compute keypair name used for CCE nodes."
  type        = string
}

variable "node_storage_kms_id" {
  description = "KMS key ID used to encrypt CCE node volumes."
  type        = string
  default     = null
}

variable "kubernetes_version_min" {
  description = "Kubernetes cluster version, for example v1.29. If null, the provider uses the current default."
  type        = string
  default     = null
}

variable "cluster_flavor_id" {
  description = "CCE cluster flavor."
  type        = string
  default     = "cce.s1.small"
}

variable "cluster_type" {
  description = "CCE cluster type."
  type        = string
  default     = "VirtualMachine"
}

variable "description" {
  description = "CCE cluster description."
  type        = string
  default     = "kubara managed CCE cluster"
}

variable "container_network_type" {
  description = "CCE container network type."
  type        = string
  default     = "overlay_l2"
}

variable "enable_public_endpoint" {
  description = "Create an EIP and bind it to the CCE master so the API server is reachable from outside the VPC. Required for running Terraform Helm/Kubernetes providers against the cluster from an external machine."
  type        = bool
  default     = true
}

variable "public_endpoint_eip_type" {
  description = "EIP type for the CCE master public endpoint."
  type        = string
  default     = "5_bgp"
}

variable "public_endpoint_eip_bandwidth_name" {
  description = "Bandwidth name for the CCE master public endpoint EIP. If empty, the cluster name is used."
  type        = string
  default     = ""
}

variable "public_endpoint_eip_bandwidth_size" {
  description = "Bandwidth size in Mbit/s for the CCE master public endpoint EIP."
  type        = number
  default     = 5
}

variable "public_endpoint_eip_bandwidth_share_type" {
  description = "Bandwidth share type for the CCE master public endpoint EIP."
  type        = string
  default     = "PER"
}

variable "public_endpoint_eip_bandwidth_charge_mode" {
  description = "Bandwidth charge mode for the CCE master public endpoint EIP."
  type        = string
  default     = "traffic"
}

variable "node_pools" {
  description = <<EOF
List of CCE node pools. Each element must be an object with:
- name               = string
- flavor             = string
- initial_node_count = number
- availability_zone  = string
- runtime            = optional(string, "containerd")
- os                 = optional(string, "EulerOS 2.9")
- scale_enable       = optional(bool, false)
- docker_base_size   = optional(number, 20)
- root_volume        = optional(object({ size = number, volumetype = string }))
- data_volumes       = optional(list(object({ size = number, volumetype = string })))
EOF
  type = list(object({
    name               = string
    flavor             = string
    initial_node_count = number
    availability_zone  = string
    runtime            = optional(string, "containerd")
    os                 = optional(string, "EulerOS 2.9")
    scale_enable       = optional(bool, false)
    docker_base_size   = optional(number, 20)
    root_volume = optional(object({
      size       = number
      volumetype = string
      }), {
      size       = 50
      volumetype = "SSD"
    })
    data_volumes = optional(list(object({
      size       = number
      volumetype = string
      })), [{
      size       = 100
      volumetype = "SSD"
    }])
  }))
  default = []
}

variable "addons" {
  description = "CCE addons keyed by addon name. Disabled addons are not managed."
  type = map(object({
    enabled = optional(bool, true)
    version = string
    basic   = optional(map(any), {})
    custom  = optional(map(any), {})
  }))
  default = {}
}

variable "create_kubeconfig_local" {
  type        = bool
  default     = false
  description = "If true, write the kubeconfig to a local file."
}

variable "kubeconfig_path" {
  type        = string
  default     = "~/.kube/config"
  description = "Filesystem path where the kubeconfig will be written if create_kubeconfig_local is true."
}

variable "kubeconfig_duration" {
  description = "Kubeconfig certificate validity in days. Use -1 for the provider maximum."
  type        = number
  default     = -1
}
