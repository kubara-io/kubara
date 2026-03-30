terraform {
  required_version = ">=1.9.3"
  required_providers {
    stackit = {
      source  = "stackitcloud/stackit"
      version = "0.88.0"
    }

    aws = {
      source  = "hashicorp/aws"
      version = ">= 6.12.0"
    }
  }
}

provider "aws" {
  region                      = var.region
  skip_credentials_validation = true
  skip_region_validation      = true
  skip_requesting_account_id  = true
  access_key                  = stackit_objectstorage_credential.this.access_key
  secret_key                  = stackit_objectstorage_credential.this.secret_access_key
  endpoints {
    s3 = "https://object.storage.eu01.onstackit.cloud"
  }
}
