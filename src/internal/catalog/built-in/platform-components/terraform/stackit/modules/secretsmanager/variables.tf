variable "project_id" {
  description = "STACKIT project ID"
  type        = string
}

variable "name" {
  description = "Name of the Secrets Manager instance"
  type        = string
}

variable "acls" {
  description = "Set of CIDR blocks allowed to access this instance"
  type        = set(string)
  default     = ["0.0.0.0/0"]
}

variable "users" {
  description = <<EOF
List of users to create.
Each element must be an object with:
- description (string): human‐readable ID for the user
- write_enabled (bool): whether this user has write access
EOF
  type = list(object({
    description   = string
    write_enabled = bool
  }))
  default = []
}
