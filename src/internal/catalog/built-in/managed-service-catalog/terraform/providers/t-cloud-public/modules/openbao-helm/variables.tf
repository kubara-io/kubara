variable "release_name" {
  description = "OpenBao Helm release name."
  type        = string
  default     = "openbao"
}

variable "namespace" {
  description = "Kubernetes namespace for OpenBao."
  type        = string
  default     = "openbao"
}

variable "repository" {
  description = "OpenBao Helm chart repository."
  type        = string
  default     = "https://openbao.github.io/openbao-helm"
}

variable "chart" {
  description = "OpenBao Helm chart name."
  type        = string
  default     = "openbao"
}

variable "chart_version" {
  description = "OpenBao Helm chart version."
  type        = string
  default     = "0.28.3"
}

variable "replicas" {
  description = "OpenBao HA replica count."
  type        = number
  default     = 3
}

variable "image_pull_secrets" {
  description = "Image pull secrets for OpenBao pods."
  type        = list(string)
  default     = []
}

variable "injector_enabled" {
  description = "Enable the OpenBao agent injector."
  type        = bool
  default     = false
}

variable "data_storage_size" {
  description = "OpenBao raft PVC size."
  type        = string
  default     = "10Gi"
}

variable "data_storage_class" {
  description = "OpenBao raft PVC storage class. Leave empty to use the cluster default."
  type        = string
  default     = ""
}

variable "seal_config" {
  description = "Optional OpenBao seal stanza appended to the HA raft config."
  type        = string
  default     = ""
}

variable "extra_environment_vars" {
  description = "Extra environment variables for the OpenBao server pods."
  type        = map(string)
  default     = {}
}
