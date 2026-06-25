# STACKIT Cloud and Edge setup

For STACKIT-based setups, kubara provides Terraform configuration and modules to provision Kubernetes and additional infrastructure like a Secret Manager instance.

!!! warning
    In kubara version `0.2.0`, Terraform generation does not merge user-customized Terraform values and will overwrite existing Terraform files.

!!! info
    You will need access to the STACKIT API. Setup instructions are available in the [Terraform provider documentation](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs) and [STACKIT Docs](https://docs.stackit.cloud/platform/access-and-identity/service-accounts/how-tos/manage-service-accounts/).
    Make sure your created Service Account has Project Owner permissions.

Follow this order:

1. Start with [Terraform Bootstrap](stackit_terraform_bootstrap.md).
2. Choose exactly one provisioning path for your cluster type: [SKE](stackit_provisioning_ske.md) or [Edge Cloud](stackit_provisioning_edgecloud.md).
3. Continue with the generic [Bootstrap Your Own Platform](../bootstrapping.md) guide.
