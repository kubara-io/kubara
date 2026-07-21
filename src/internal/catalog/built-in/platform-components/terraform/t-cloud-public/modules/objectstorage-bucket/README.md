# Object storage bucket

Creates a T Cloud Public OBS bucket plus a dedicated identity user and access key/secret key pair for S3-compatible access.

The bucket policy grants bucket metadata access, including `HeadBucket`, because S3-compatible clients commonly validate buckets before reading or writing objects.

The generated bootstrap example uses this module for Terraform state storage.
