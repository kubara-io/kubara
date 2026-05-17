terraform {
  required_version = ">=1.9.3"
  required_providers {
    stackit = {
      source  = "stackitcloud/stackit"
      version = "0.96.0"
    }
    time = {
      source  = "hashicorp/time"
      version = "0.14.0"
    }
  }
}
