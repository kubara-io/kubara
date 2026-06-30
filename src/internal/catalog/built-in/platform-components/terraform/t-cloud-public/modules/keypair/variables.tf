variable "name" {
  description = "Name of the compute keypair."
  type        = string
}

variable "algorithm" {
  description = "Private key algorithm."
  type        = string
  default     = "ED25519"
}
