# OpenBao Helm

Deploys OpenBao into a CCE cluster with the official OpenBao Helm chart.

The module enables HA mode with integrated Raft storage. The optional `seal_config` input appends a supported OpenBao seal stanza to the generated Raft configuration.
The Helm release does not wait for readiness because the default flow initializes and unseals OpenBao manually after installation.

When `ingress_enabled` is set, the module exposes OpenBao below the configured path prefix and creates the Traefik rewrite middlewares required for `/openbao`.
