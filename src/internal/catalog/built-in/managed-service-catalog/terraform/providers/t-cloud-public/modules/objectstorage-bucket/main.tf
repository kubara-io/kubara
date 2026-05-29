resource "opentelekomcloud_identity_user_v3" "this" {
  name        = var.user_name
  pwd_reset   = false
  description = "Created by kubara Terraform for OBS bucket access"
}

resource "opentelekomcloud_identity_credential_v3" "this" {
  user_id     = opentelekomcloud_identity_user_v3.this.id
  description = "Created by kubara Terraform for OBS bucket access"
}

resource "opentelekomcloud_identity_group_v3" "obs_only" {
  name        = "${var.user_name}-obs-only"
  description = "Base group for ${var.bucket_name} OBS bucket user"
}

resource "opentelekomcloud_identity_user_group_membership_v3" "obs_only" {
  user   = opentelekomcloud_identity_user_v3.this.id
  groups = [opentelekomcloud_identity_group_v3.obs_only.id]
}

data "opentelekomcloud_identity_project_v3" "region_project" {
  provider = opentelekomcloud.global-region
}

resource "opentelekomcloud_identity_group_v3" "kms_access" {
  count = var.enable_server_side_encryption ? 1 : 0

  name        = "${var.user_name}-obs-kms-rw"
  description = "KMS access for ${var.bucket_name} OBS bucket user"
}

resource "opentelekomcloud_identity_role_v3" "kms_access" {
  count = var.enable_server_side_encryption ? 1 : 0

  description   = "KMS access for ${var.bucket_name} OBS bucket user"
  display_name  = "${var.user_name}-obs-kms-rw"
  display_layer = "project"

  statement {
    effect = "Allow"
    action = [
      "kms:cmk:get",
      "kms:grant:list",
      "kms:cmk:getMaterial",
      "kms:cmk:getRotation",
      "kms:cmk:deleteMaterial",
      "kms:cmk:disable",
      "kms:dek:crypto",
      "kms:cmk:disableRotation",
      "kms:cmk:importMaterial",
      "kms:grant:retire",
      "kms:cmkTag:create",
      "kms:dek:create",
      "kms:grant:revoke",
      "kms:cmk:updateRotation",
      "kms:grant:create",
      "kms:cmk:update",
      "kms:cmk:generate",
      "kms:cmkTag:batch",
      "kms:cmk:enableRotation",
      "kms:cmkTag:delete",
      "kms:cmk:enable",
      "kms:cmk:crypto"
    ]
    resource = ["kms:*:*:KeyId:${var.kms_key_id}"]
  }
}

resource "opentelekomcloud_identity_role_assignment_v3" "kms_access" {
  count = var.enable_server_side_encryption ? 1 : 0

  group_id   = opentelekomcloud_identity_group_v3.kms_access[0].id
  project_id = data.opentelekomcloud_identity_project_v3.region_project.id
  role_id    = opentelekomcloud_identity_role_v3.kms_access[0].id
}

resource "opentelekomcloud_identity_user_group_membership_v3" "kms_access" {
  count = var.enable_server_side_encryption ? 1 : 0

  user   = opentelekomcloud_identity_user_v3.this.id
  groups = [opentelekomcloud_identity_group_v3.kms_access[0].id]
}

resource "opentelekomcloud_obs_bucket" "this" {
  bucket      = var.bucket_name
  acl         = var.acl
  versioning  = var.versioning
  parallel_fs = var.parallel_fs

  dynamic "server_side_encryption" {
    for_each = var.enable_server_side_encryption ? [1] : []
    content {
      algorithm  = "kms"
      kms_key_id = var.kms_key_id
    }
  }

  lifecycle {
    precondition {
      condition     = !var.enable_server_side_encryption || var.kms_key_id != null
      error_message = "kms_key_id must be set when enable_server_side_encryption is true."
    }
  }
}

resource "opentelekomcloud_obs_bucket_policy" "this" {
  bucket = opentelekomcloud_obs_bucket.this.id
  policy = <<POLICY
{
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "ID": ["domain/${opentelekomcloud_identity_user_v3.this.domain_id}:user/${opentelekomcloud_identity_user_v3.this.id}"]
      },
      "Action": [
        "ListBucket",
        "GetBucketLocation",
        "ListBucketMultipartUploads"
      ],
      "Resource": [
        "${opentelekomcloud_obs_bucket.this.bucket}"
      ]
    },
    {
      "Effect": "Allow",
      "Principal": {
        "ID": ["domain/${opentelekomcloud_identity_user_v3.this.domain_id}:user/${opentelekomcloud_identity_user_v3.this.id}"]
      },
      "Action": [
        "GetObject",
        "PutObject",
        "DeleteObject",
        "GetObjectVersion",
        "DeleteObjectVersion",
        "ListMultipartUploadParts",
        "AbortMultipartUpload"
      ],
      "Resource": [
        "${opentelekomcloud_obs_bucket.this.bucket}/*"
      ]
    }
  ]
}
POLICY
}
