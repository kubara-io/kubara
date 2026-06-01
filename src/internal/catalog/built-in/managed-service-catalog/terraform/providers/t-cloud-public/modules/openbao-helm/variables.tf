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

variable "create_namespace" {
  description = "Create the OpenBao namespace through the Helm release."
  type        = bool
  default     = true
}

variable "helm_timeout" {
  description = "Helm release timeout in seconds."
  type        = number
  default     = 900
}

variable "image_registry" {
  description = "OpenBao server image registry."
  type        = string
  default     = "quay.io"
}

variable "image_repository" {
  description = "OpenBao server image repository."
  type        = string
  default     = "openbao/openbao"
}

variable "image_tag" {
  description = "OpenBao server image tag. Leave empty to use the chart appVersion."
  type        = string
  default     = ""
}

variable "image_pull_policy" {
  description = "OpenBao server image pull policy."
  type        = string
  default     = "IfNotPresent"
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
  default     = true
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

variable "extra_secret_environment_vars" {
  description = "Extra environment variables sourced from existing Kubernetes Secrets."
  type = list(object({
    envName    = string
    secretName = string
    secretKey  = string
  }))
  default = []
}
