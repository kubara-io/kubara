terraform {
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "< 4.0.0"
    }
  }
}
