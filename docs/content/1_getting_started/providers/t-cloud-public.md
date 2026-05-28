# T Cloud Public setup

T Cloud Public is supported through provider-specific Terraform templates.

The current kubara provider key is `t-cloud-public` and the Kubernetes type is `cce` for Cloud Container Engine.

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

`projectId` is used as the generated default for `t_cloud_public_tenant_name`. You can override it with `TF_VAR_t_cloud_public_tenant_name` if your tenant naming differs.

Follow this order:

1. Start with [Terraform Bootstrap](t-cloud-public_terraform_bootstrap.md).
2. Continue with [Provisioning Infrastructure (CCE)](t-cloud-public_provisioning_cce.md).
3. Continue with the generic [Bootstrap Your Own Platform](../bootstrapping.md) guide.

## Provider References

- [T Cloud Public Terraform provider](https://registry.terraform.io/providers/opentelekomcloud/opentelekomcloud/latest/docs)
- [CCE cluster resource](https://registry.terraform.io/providers/opentelekomcloud/opentelekomcloud/latest/docs/resources/cce_cluster_v3)
- [CCE kubeconfig data source](https://registry.terraform.io/providers/opentelekomcloud/opentelekomcloud/latest/docs/data-sources/cce_cluster_kubeconfig_v3)
