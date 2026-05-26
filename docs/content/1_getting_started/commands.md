# kubara commands

# NAME

kubara - Opinionated CLI for Kubernetes platform engineering

# SYNOPSIS

kubara

```
[--base64]
[--catalog-overwrite]
[--catalog]=[value]
[--check-update]
[--config-file|-c]=[value]
[--decode]
[--encode]
[--env-file]=[value]
[--file]=[value]
[--help|-h]
[--kubeconfig]=[value]
[--string]=[value]
[--test-connection]
[--version|-v]
[--work-dir|-w]=[value]
```

# DESCRIPTION

kubara is an opinionated CLI to bootstrap and operate Kubernetes platforms with GitOps-first workflows.

**Usage**:

```
kubara [GLOBAL OPTIONS] [command [COMMAND OPTIONS]] [ARGUMENTS...]
```

# GLOBAL OPTIONS

**--base64**: Enable base64 encode/decode mode

**--catalog**="": Path to external ServiceDefinition catalog directory.

**--catalog-overwrite**: Allow external service definitions from --catalog to overwrite built-in definitions on name collisions.

**--check-update**: Check online for a newer kubara release

**--config-file, -c**="": Path to the configuration file (default: "config.yaml")

**--decode**: Base64 decode input

**--encode**: Base64 encode input

**--env-file**="": Path to the .env file (default: ".env")

**--file**="": Input file path for base64 operation

**--help, -h**: show help

**--kubeconfig**="": Path to kubeconfig file (default: "~/.kube/config")

**--string**="": Input string for base64 operation

**--test-connection**: Check if Kubernetes cluster can be reached. List namespaces and exit

**--version, -v**: print the version

**--work-dir, -w**="": Working directory (default: ".")


# COMMANDS

## init

Initialize a new kubara directory

**--envVarPrefix**="": Prefix for envs read from envVars (default: "KUBARA_")

**--help, -h**: show help

**--overwrite**: Overwrite config if exists

**--prep**: Copy embedded prep/ folder into current working directory

### help, h

Shows a list of commands or help for one command

## generate

generates files from embedded templates and the config file; by default for both Helm and Terraform

>generate [--terraform|--helm] [--managed-catalog <path> --overlay-values <path>] [--catalog <path> [--catalog-overwrite]] [--dry-run]

**--dry-run**: Preview generation without creating files

**--helm**: Only generate Helm files

**--help, -h**: show help

**--managed-catalog**="": Path to the managed catalog directory. (default: "managed-service-catalog")

**--overlay-values**="": Path to overlay values directory. (default: "customer-service-catalog")

**--terraform**: Only generate Terraform files

### help, h

Shows a list of commands or help for one command

## bootstrap

Bootstrap ArgoCD onto the specified cluster with optional external-secrets and prometheus CRD

**--dry-run**: Run with dry-run

**--envVarPrefix**="": Prefix for envs read from envVars (default: "KUBARA_")

**--help, -h**: show help

**--managed-catalog**="": Path to the managed catalog directory (default: "managed-service-catalog")

**--overlay-values**="": Path to overlay values directory (default: "customer-service-catalog")

**--timeout**="": Timeout for kubernetes API calls (e.g. 10s, 1m) (default: 5m0s)

**--with-es-crds**: Also install external-secrets

**--with-es-css-file**="": Path to the ClusterSecretStore manifest file (supports go-template + sprig)

**--with-prometheus-crds**: Also install kube-prometheus-stack

### help, h

Shows a list of commands or help for one command

## schema

Generate JSON schema file for config structure

>schema [--output] [--catalog <path> [--catalog-overwrite]]

**--help, -h**: show help

**--output, -o**="": Output file path for the JSON schema (default: "config.schema.json")

### help, h

Shows a list of commands or help for one command

## help, h

Shows a list of commands or help for one command
