# CCE cluster

Creates a T Cloud Public CCE cluster and node pools.

Inputs such as VPC ID, subnet ID, node SSH keypair name, and node volume KMS key ID are supplied by separate modules or existing infrastructure. This keeps the cluster lifecycle separate from shared network and security primitives.

The module keeps the cluster lifecycle separate from shared network and security primitives so the generated customer stack can compose infrastructure in small steps.
