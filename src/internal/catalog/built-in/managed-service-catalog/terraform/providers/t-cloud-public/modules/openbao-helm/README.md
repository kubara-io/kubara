# OpenBao Helm

Deploys OpenBao into a CCE cluster with the official OpenBao Helm chart.

The module enables HA mode with integrated Raft storage. The optional `seal_config` input can append a supported OpenBao seal stanza to the generated Raft configuration. T Cloud Public KMS is not a native OpenBao seal today, so KMS auto-unseal requires a supported external seal or an installed OpenBao KMS plugin.
