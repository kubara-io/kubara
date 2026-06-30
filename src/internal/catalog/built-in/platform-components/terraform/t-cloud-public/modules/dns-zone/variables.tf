variable "name" {
  description = "DNS zone name. A trailing dot is added automatically if omitted."
  type        = string
}

variable "email" {
  description = "Administrative email contact for the DNS zone."
  type        = string
}

variable "type" {
  description = "DNS zone type."
  type        = string
  default     = "public"

  validation {
    condition     = contains(["public", "private"], var.type)
    error_message = "Allowed values for type are \"public\" and \"private\"."
  }
}

variable "ttl" {
  description = "Default DNS zone TTL in seconds."
  type        = number
  default     = 300
}

variable "description" {
  description = "DNS zone description."
  type        = string
  default     = "kubara managed DNS zone"
}

variable "tags" {
  description = "Tags assigned to the DNS zone."
  type        = map(string)
  default     = {}
}

variable "routers" {
  description = "Router attachments for private DNS zones."
  type = list(object({
    router_id     = string
    router_region = string
  }))
  default = []
}
