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
  description = "OpenBao raft PVC size. OpenBao only persists Raft state and KV data, so 1 GiB is plenty for typical kubara platform secret workloads."
  type        = string
  default     = "1Gi"
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

variable "ingress_enabled" {
  description = "Expose OpenBao through an ingress."
  type        = bool
  default     = false
}

variable "ingress_host" {
  description = "OpenBao ingress host."
  type        = string
  default     = ""

  validation {
    condition     = !var.ingress_enabled || var.ingress_host != ""
    error_message = "ingress_host must be set when ingress_enabled is true."
  }
}

variable "ingress_path" {
  description = "Path prefix for the OpenBao ingress."
  type        = string
  default     = "/openbao"

  validation {
    condition     = startswith(var.ingress_path, "/")
    error_message = "ingress_path must start with '/'."
  }
}

variable "ingress_class_name" {
  description = "IngressClass name used for the OpenBao ingress."
  type        = string
  default     = ""
}

variable "ingress_annotations" {
  description = "Additional OpenBao ingress annotations. Values override module defaults on key conflicts."
  type        = map(string)
  default     = {}
}

variable "ingress_tls_secret_name" {
  description = "TLS secret name for the OpenBao ingress. Leave empty to omit TLS."
  type        = string
  default     = ""
}

variable "ingress_create_traefik_middlewares" {
  description = "Create the Traefik Middleware objects for the OpenBao subpath ingress. Requires the traefik.io/v1alpha1 CRD to already exist in the cluster. Set to true on a follow-up apply once Traefik is installed."
  type        = bool
  default     = false
}
