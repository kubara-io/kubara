resource "stackit_image" "this" {
  project_id      = var.project_id
  name            = var.name
  disk_format     = var.disk_format
  local_file_path = var.local_file_path
  min_disk_size   = var.min_disk_size
  config          = var.config
}
