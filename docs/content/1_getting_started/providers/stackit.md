# STACKIT Cloud and Edge setup

We recommend using Terraform to provision your Kubernetes and additional infrastructure like a Secret Manager instance.
For this purpose we provide the necessary Terraform configuration and modules.

Follow this order:

1. Start with [Terraform Bootstrap](stackit_terraform_bootstrap.md).
2. Choose exactly one provisioning path for your cluster type: [SKE](stackit_provisioning_ske.md) or [Edge Cloud](stackit_provisioning_edgecloud.md).
3. Continue with the generic [Bootstrap Your Own Platform](../bootstrapping.md) guide.
