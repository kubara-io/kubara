# OpenBao Helm

Deploys OpenBao into a CCE cluster with the official OpenBao Helm chart.

The module enables HA mode with integrated Raft storage. The optional `seal_config` input appends a supported OpenBao seal stanza to the generated Raft configuration. `extra_secret_environment_vars` can reference Kubernetes Secrets for seal credentials, and the image inputs allow testing custom OpenBao builds while T Cloud Public KMS support is validated.
