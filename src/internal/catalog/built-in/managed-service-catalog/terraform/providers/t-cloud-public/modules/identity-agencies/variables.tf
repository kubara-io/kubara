variable "agencies" {
  description = "IAM agencies keyed by logical name."
  type = map(object({
    name                  = string
    description           = optional(string)
    delegated_domain_name = string
    project_roles = optional(list(object({
      project      = optional(string)
      all_projects = optional(bool)
      roles        = list(string)
    })), [])
    domain_roles = optional(list(string), [])
  }))
}
