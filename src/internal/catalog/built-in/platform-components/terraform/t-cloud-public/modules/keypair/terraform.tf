terraform {
  required_version = ">= 1.9.3"

  required_providers {
    opentelekomcloud = {
      source  = "opentelekomcloud/opentelekomcloud"
      version = ">= 1.36.64, < 2.0.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = ">= 4.3.0, < 5.0.0"
    }
  }
}
