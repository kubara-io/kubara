resource "stackit_image" "this" {
  project_id      = var.project_id
  name            = var.name
  disk_format     = var.disk_format
  local_file_path = var.local_file_path
  min_disk_size   = var.min_disk_size
  config = {
    operating_system         = var.operating_system
    operating_system_distro  = var.operating_system_distro
    operating_system_version = var.operating_system_version
  }
}
