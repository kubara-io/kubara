output "name" {
  description = "Compute keypair name."
  value       = opentelekomcloud_compute_keypair_v2.this.name
}

output "public_key_openssh" {
  description = "OpenSSH public key."
  value       = tls_private_key.this.public_key_openssh
}

output "private_key_openssh" {
  description = "OpenSSH private key."
  value       = tls_private_key.this.private_key_openssh
  sensitive   = true
}
