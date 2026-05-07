<p align="center">
  <img src="docs/content/images/logo.svg" alt="kubara logo" width="180" />
</p>

# kubara

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](./LICENSE)
[![Docs License](https://img.shields.io/badge/docs%20license-CC%20BY%204.0-2ea44f.svg)](./NOTICE.md#documentation-license)
[![Docs](https://img.shields.io/badge/docs-docs.kubara.io-1f6feb)](https://docs.kubara.io)
[![Slack](https://img.shields.io/badge/slack-Kubernetes_%23kubara-red?logo=slack)](https://app.slack.com/client/T09NY5SBT/C0B2X0DLPR6)
[![Codeberg Mirror](https://img.shields.io/badge/codeberg-kubara--io%2Fkubara-blue)](https://codeberg.org/kubara-io/kubara)
[![FOSSA License Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkubara-io%2Fkubara.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkubara-io%2Fkubara?ref=badge_shield)
[![FOSSA Security Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkubara-io%2Fkubara.svg?type=shield&issueType=security)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkubara-io%2Fkubara?ref=badge_shield&issueType=security)

kubara is an opinionated CLI to bootstrap and operate Kubernetes platforms with a GitOps-first workflow.
It combines platform scaffolding, environment configuration, and production-ready defaults in a single binary.

## Why kubara

- One CLI for platform setup and lifecycle tasks
- GitOps-native structure for repeatable deployments
- Built for multi-cluster and multi-tenant environments
- Extensible with Terraform and Helm based components

## Installation

See [INSTALLATION.md](docs/content/1_getting_started/installation.md) for Linux, macOS, and Windows installation instructions.

## Documentation

- Public docs: <https://docs.kubara.io>
- Local docs sources: [`docs/`](./docs)

## CLI Commands

```text
init       Initialize a new kubara directory
generate   generates files from embedded templates and config (default: both Helm and Terraform).
bootstrap  Bootstrap ArgoCD onto the specified cluster with optional external-secrets and prometheus CRD
schema     Generate JSON schema file for config structure
help, h    Shows a list of commands or help for one command
```

## Global Options

```text
--kubeconfig string               Path to kubeconfig file (default: "~/.kube/config")
--work-dir string, -w string      Working directory (default: ".")
--config-file string, -c string   Path to the configuration file (default: "config.yaml")
--env-file string                 Path to the .env file (default: ".env")
--test-connection                 Check if Kubernetes cluster can be reached. List namespaces and exit
--base64                          Enable base64 encode/decode mode
--encode                          Base64 encode input
--decode                          Base64 decode input
--string string                   Input string for base64 operation
--file string                     Input file path for base64 operation
--check-update                    Check online for a newer kubara release
--help, -h                        show help
--version, -v                     print the version
```

## Update Check

- kubara checks for newer GitHub releases on each run; disable with `KUBARA_UPDATE_CHECK=0`; run `kubara --check-update` for a live check.

## Community and Support

Join the #kubara [Slack channel](https://app.slack.com/client/T09NY5SBT/C0B2X0DLPR6) to chat with other users of kubara or reach out to the maintainers directly. Use the [public invite link](http://slack.k8s.io/) (http://slack.k8s.io/) to get an invite for the official Kubernetes Slack space.

- Questions and bug reports: [GitHub Issues](https://github.com/kubara-io/kubara/issues)
- Discussions and Q&A: [GitHub Discussions](https://github.com/kubara-io/kubara/discussions)
- Team and contributor guidance: [CONTRIBUTING.md](./CONTRIBUTING.md)
- Code of conduct: [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)

## Contributing

Contributions are welcome.
Please read [CONTRIBUTING.md](./CONTRIBUTING.md) before opening a pull request.

## License

kubara uses dual licensing:

- Software source code: [Apache 2.0](./LICENSE)
- Documentation: [CC BY 4.0](./NOTICE.md#documentation-license)
- Additional notices: [NOTICE.md](./NOTICE.md)


[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fkubara-io%2Fkubara.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fkubara-io%2Fkubara?ref=badge_large)
