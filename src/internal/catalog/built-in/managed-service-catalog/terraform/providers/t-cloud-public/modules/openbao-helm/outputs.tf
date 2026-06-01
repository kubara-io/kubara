output "release_name" {
  description = "OpenBao Helm release name."
  value       = helm_release.this.name
}

output "namespace" {
  description = "OpenBao namespace."
  value       = helm_release.this.namespace
}

output "chart_version" {
  description = "OpenBao chart version."
  value       = helm_release.this.version
}

output "ingress_url" {
  description = "OpenBao ingress URL, if enabled."
  value       = local.ingress_url
}
