terraform {
  required_version = ">= 1.9.3"

  required_providers {
    opentelekomcloud = {
      source  = "opentelekomcloud/opentelekomcloud"
      version = ">= 1.36.64, < 2.0.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.9.0, < 4.0.0"
    }
  }
}
