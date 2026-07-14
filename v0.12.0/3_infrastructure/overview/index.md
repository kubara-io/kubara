# Infrastructure Presets

kubara is first and foremost a **bootstrapping and platform building tool on top of Kubernetes**.
It helps you generate, bootstrap, and shape the platform layer that runs on a Kubernetes cluster.

kubara can run on **any Kubernetes environment**. The pages in this section are infrastructure **presets** and example flows we provide as convenient starting points, not as a hard provider limit.

They are a nice perk, but they are **not required** to use kubara.
If your environment gives kubara the same core capabilities, you can skip these presets and use the normal bootstrap flow.

## What kubara expects from the infrastructure

| Capability                                                                                                   | Why kubara needs it                                                                   |
| ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------- |
| Kubernetes cluster access                                                                                    | kubara bootstraps Argo CD and the platform services into a target cluster             |
| Secret backend supported by [external-secrets](https://external-secrets.io/latest/provider/hashicorp-vault/) | Platform components read secrets through `ClusterSecretStore` / `ExternalSecret`      |
| DNS automation for ingress hosts                                                                             | kubara defaults are built around `external-dns` managing records for exposed services |
| Ingress entrypoint                                                                                           | Platform UIs and apps need HTTP/HTTPS entrypoints                                     |
| Optional object storage                                                                                      | Only needed when you enable Velero bucket creation                                    |

In practice this means:

1. Your cluster must be reachable with a kubeconfig.
2. You need a secret manager or vault that `external-secrets` can read from.
3. You should have a DNS zone that `external-dns` can manage automatically, or you need to replace that part with your own process.
4. You need an ingress setup that matches your `ingressClassName` and service annotations.

## Available presets

- [kind (local)](kind.md): the local evaluation setup used by `kubara bootstrap --local`, including `cloud-provider-kind`, Traefik, and OpenBao in dev mode.
- [STACKIT SKE](stackit_ske.md): managed Kubernetes on STACKIT with the generated Terraform modules for DNS, Secrets Manager, IAM, and the cluster itself.
- [STACKIT Edge Cloud](stackit_edge_cloud.md): an edge-focused flow where kubara generates the Terraform and you finish the Edge-specific image and cluster steps in STACKIT Edge Cloud.

If you already have your own Kubernetes cluster and equivalent infrastructure, continue directly with the generic [Bootstrap Your Own Platform](../1_getting_started/bootstrapping.md) guide.
