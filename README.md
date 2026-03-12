# Kubara

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](./LICENSE)
[![Docs License](https://img.shields.io/badge/docs%20license-CC%20BY%204.0-2ea44f.svg)](./LICENSE-docs)
[![Docs](https://img.shields.io/badge/docs-docs.kubara.io-1f6feb)](https://docs.kubara.io)

Kubara is an opinionated CLI to bootstrap and operate Kubernetes platforms with a GitOps-first workflow.
It combines platform scaffolding, environment configuration, and production-ready defaults in a single binary.

## Why Kubara

- One CLI for platform setup and lifecycle tasks
- GitOps-native structure for repeatable deployments
- Built for multi-cluster and multi-tenant environments
- Extensible with Terraform and Helm based components

## Installation

See [INSTALLATION.md](./INSTALLATION.md) for Linux, macOS, and Windows installation instructions.

## Documentation

- Public docs: <https://docs.kubara.io>
- Local docs sources: [`docs/`](./docs)

## CLI Commands

```text
init       Initialize a new kubara directory
generate   generates files from embedded templates and config.
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
--help, -h                        show help
--version, -v                     print the version
```

## Community and Support

- Questions and bug reports: [GitHub Issues](https://github.com/kubara-io/kubara/issues)
- Discussions and Q&A: [GitHub Discussions](https://github.com/kubara-io/kubara/discussions)
- Team and contributor guidance: [CONTRIBUTING.md](./CONTRIBUTING.md)
- Code of conduct: [docs/content/6_reference/code_of_conduct.md](./docs/content/6_reference/code_of_conduct.md)

## Contributing

Contributions are welcome.
Please read [CONTRIBUTING.md](./CONTRIBUTING.md) before opening a pull request.

## License

Kubara uses dual licensing:

- Software source code: [Apache 2.0](./LICENSE)
- Documentation: [CC BY 4.0](./LICENSE-docs)
- Additional notices: [NOTICE.txt](./NOTICE.txt)
