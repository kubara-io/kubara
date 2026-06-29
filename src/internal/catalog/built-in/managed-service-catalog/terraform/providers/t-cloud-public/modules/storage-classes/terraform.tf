terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "< 4.0.0"
    }
  }
}
