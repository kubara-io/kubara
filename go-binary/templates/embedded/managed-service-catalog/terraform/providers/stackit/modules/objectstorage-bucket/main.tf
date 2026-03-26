# 1) Create a credentials group
resource "stackit_objectstorage_credentials_group" "this" {
  project_id = var.project_id
  name       = var.credentials_group_name
  region     = var.region
}

# 2) Create the bucket, only after the group exists
resource "stackit_objectstorage_bucket" "this" {
  project_id = var.project_id
  name       = var.bucket_name
  region     = var.region

  depends_on = [stackit_objectstorage_credentials_group.this]
}

# 3) Create one credential in that group
resource "stackit_objectstorage_credential" "this" {
  project_id           = var.project_id
  credentials_group_id = stackit_objectstorage_credentials_group.this.credentials_group_id
  region               = var.region

  depends_on = [stackit_objectstorage_credentials_group.this]
}


# Allow only terraform credentials group to access the tfstate bucket
# WARNING:
# Be careful when attaching a restrictive bucket policy!
# If the credentials group referenced in the policy are deleted or lost,
# you will no longer be able to access, update, or delete the bucket — not even via Terraform.
# In such a case, manual recovery through a Stackit support ticket will be required.
#
# resource "aws_s3_bucket_policy" "this" {
#   bucket = stackit_objectstorage_bucket.this.name
#   policy = <<EOF
#   {
#     "Statement": [
#       {
#         "Sid": "allow-specific-credential-group",
#         "Effect": "Deny",
#         "NotPrincipal": {
#           "AWS": "${stackit_objectstorage_credentials_group.this.urn}"
#         },
#         "Action": [
#           "s3:*"
#         ],
#         "Resource": [
#           "arn:aws:s3:::${stackit_objectstorage_bucket.this.name}",
#           "arn:aws:s3:::${stackit_objectstorage_bucket.this.name}/*"
#         ]
#       }
#     ]
#   }
#   EOF
#
#   depends_on = [stackit_objectstorage_bucket.this, stackit_objectstorage_credentials_group.this]
# }
