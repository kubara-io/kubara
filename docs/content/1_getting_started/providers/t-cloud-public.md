# T Cloud Public (Community)

The terraform modules for the T Cloud Public are built by the kubara community and aren't tested on a regular basis through integration nor regression tests by the kubara maintainers.

The kubara provider key is `t-cloud-public` and the Kubernetes type is `cce` for Cloud Container Engine.

## Configuration

Use these values in `config.yaml`:

```yaml
terraform:
  provider: t-cloud-public
  projectId: <tenant-name>
  kubernetesType: cce
  kubernetesVersion: 1.29
  dns:
    name: <dns-name>
    email: <email>
```

For T Cloud Public, set `projectId` to the tenant/project name used as `tenant_name`, not to a UUID.

## Generated Terraform

Running `kubara generate --terraform` creates a T Cloud Public Terraform layout with:

- `bootstrap-tfstate-backend`: OBS bucket, backend credentials, and optional OBS KMS agency setup for Terraform state
- `infrastructure`: DNS zone, IAM agencies, VPC/subnet/NAT/load balancer, keypair, KMS key, CCE cluster, and StorageClass resources
- reusable managed modules for OBS buckets, IAM agencies, DNS, network, keypair, KMS, CCE, and StorageClasses

Provider-specific platform wiring is handled separately from these Terraform core templates.
