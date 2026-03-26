terraform {
  required_version = ">=1.9.3"
  required_providers {
    stackit = {
      source  = "stackitcloud/stackit"
      version = "0.90.0"
    }
    time = {
      source  = "hashicorp/time"
      version = "0.13.1"
    }
  }
}
