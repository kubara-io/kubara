
variable "project_id" {
  description = "STACKIT project ID to associate the DNS zone with"
  type        = string
}

variable "name" {
  description = "User-defined name of the DNS Zone"
  type        = string
}


variable "dns_name" {
  description = "DNS zone name (e.g. example). You have the option to choose between: .run.onstackit.cloud, .stackit.gg, .stackit.rocks, .stackit.run or .stackit.zone"
  type        = string

}

variable "contact_email" {
  description = "Contact e-mail for the zone"
  type        = string
  default     = "hostmaster@stackit.cloud"
}

variable "acl" {
  description = "Access control list (CIDR), e.g. 0.0.0.0/0"
  type        = string
  default     = "0.0.0.0/0"
}

variable "default_ttl" {
  description = "Default time to live in seconds"
  type        = number
  default     = 300
}

variable "description" {
  description = "DNS Zone for managing services"
  type        = string
  default     = "DNS Zone for managing services"
}

variable "type" {
  description = "Zone type: primary or secondary"
  type        = string
  default     = "primary"
}

variable "is_reverse_zone" {
  description = "Whether this is a reverse lookup zone"
  type        = bool
  default     = false
}

variable "active" {
  description = "Whether the zone is active"
  type        = bool
  default     = true
}


variable "negative_cache" {
  description = "Negative caching in seconds"
  type        = number
  default     = 60
}


variable "refresh_time" {
  description = "Refresh time in seconds"
  type        = number
  default     = 3600
}

variable "retry_time" {
  description = "Retry time in seconds"
  type        = number
  default     = 600
}
