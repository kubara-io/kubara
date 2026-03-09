terraform {
  required_version = ">=1.9.3"
  required_providers {
    stackit = {
      source  = "stackitcloud/stackit"
      version = "0.85.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "2.7.0"
    }
  }
}
