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

Follow this order:

1. Start with [Terraform Bootstrap](t-cloud-public_terraform_bootstrap.md).
2. Continue with [Provisioning Infrastructure (CCE)](t-cloud-public_provisioning_cce.md).
3. Continue with the generic [Bootstrap Your Own Platform](../bootstrapping.md) guide.

## Provider References

- [T Cloud Public Terraform provider](https://registry.terraform.io/providers/opentelekomcloud/opentelekomcloud/latest/docs)
- [CCE cluster resource](https://registry.terraform.io/providers/opentelekomcloud/opentelekomcloud/latest/docs/resources/cce_cluster_v3)
- [CCE kubeconfig data source](https://registry.terraform.io/providers/opentelekomcloud/opentelekomcloud/latest/docs/data-sources/cce_cluster_kubeconfig_v3)
