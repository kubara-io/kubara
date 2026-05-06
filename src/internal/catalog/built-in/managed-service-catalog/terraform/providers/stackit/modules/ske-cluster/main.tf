resource "stackit_ske_cluster" "this" {
  project_id             = var.project_id
  name                   = var.name
  region                 = var.region
  kubernetes_version_min = var.kubernetes_version_min

  node_pools  = var.node_pools
  maintenance = var.ske_maintenance

  # The SKE API uses the name as the Cluster ID therefore any
  # change to the naming scheme or cluster name causes a full
  # disruptive recreate of the SKE cluster. Which will cause
  # data loss. Therefore we enforce that name changes in the
  # kubara config or naming scheme logic is not propagated to
  # existing clusters to ensure a stable platform.
  lifecycle {
    ignore_changes = [
      name
    ]
  }
}

resource "stackit_ske_kubeconfig" "this" {
  project_id   = var.project_id
  cluster_name = stackit_ske_cluster.this.name
  refresh      = var.refresh
  expiration   = var.expiration
}

resource "local_file" "kubeconfig" {
  count           = var.create_kubeconfig_local ? 1 : 0
  content         = stackit_ske_kubeconfig.this.kube_config
  filename        = var.kubeconfig_path
  file_permission = "0644"

  depends_on = [stackit_ske_kubeconfig.this]
}
