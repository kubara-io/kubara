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
kubara [command]
```

# GLOBAL OPTIONS

**--base64**: Enable base64 encode/decode mode

**--catalog**="": Path to an external catalog directory or an OCI reference in the form oci://registry/repository:x.y.z

**--catalog-overwrite**: Allow external service definitions from --catalog to overwrite built-in definitions on name collisions

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

Initialize kubara config for your GitOps repository

>kubara init [--prep] [--local]

**--envVarPrefix**="": Prefix for envs read from envVars (default: "KUBARA_")

**--help, -h**: show help

**--local**: Initialize files for the local evaluation workflow. Local testing only; not for production use.

**--overwrite**: Overwrite config if exists

**--prep**: Copy embedded prep/ folder into current working directory

### help, h

Shows a list of commands or help for one command

## generate

Generate files from catalog templates

>kubara generate [--terraform|--helm] [--catalog PATH_OR_OCI [--catalog-overwrite]] [--dry-run]

**--dry-run**: Preview generation without creating files

**--helm**: Only generate Helm files

**--help, -h**: show help

**--terraform**: Only generate Terraform files

### help, h

Shows a list of commands or help for one command

## bootstrap

Bootstrap Argo CD onto a cluster

>kubara bootstrap CLUSTER_NAME [--local]

**--dry-run**: Run with dry-run

**--envVarPrefix**="": Prefix for envs read from envVars (default: "KUBARA_")

**--help, -h**: show help

**--local**: Provision an isolated local evaluation environment. Local testing only; not for production use.

**--platform-components**="": Path to the platform-components directory (default: "platform-components")

**--platform-configs**="": Path to platform-configs directory (default: "platform-configs")

**--timeout**="": Timeout for kubernetes API calls (e.g. 10s, 1m) (default: 5m0s)

**--with-es-crds**: Also install external-secrets

**--with-es-css-file**="": Path to the ClusterSecretStore manifest file (supports go-template + sprig)

**--with-prometheus-crds**: Also install kube-prometheus-stack

### help, h

Shows a list of commands or help for one command

## schema

Generate a JSON schema for the config yaml structure

>kubara schema [--output PATH] [--catalog PATH_OR_OCI [--catalog-overwrite]]

**--help, -h**: show help

**--output, -o**="": Output file path for the JSON schema (default: "config.schema.json")

### help, h

Shows a list of commands or help for one command

## agents

Scaffold an onboarding file for AI coding assistants (AGENTS.md)

>kubara agents [--overwrite]

**--help, -h**: show help

**--overwrite**: Overwrite an existing AGENTS.md

### help, h

Shows a list of commands or help for one command

## catalog

Manage platform catalogs

>kubara catalog [command]

**--help, -h**: show help

### create

Create a custom catalog directory skeleton

>kubara catalog create CATALOG_NAME

**--help, -h**: show help

#### help, h

Shows a list of commands or help for one command

### add

Add a service definition to the current catalog

>kubara catalog add SERVICE_NAME

**--help, -h**: show help

#### help, h

Shows a list of commands or help for one command

### pull

Pull a catalog from a remote registry

>kubara catalog pull [--insecure] oci://registry/repository:x.y.z

**--help, -h**: show help

**--insecure**: Ignore TLS certificate verification issues for the registry connection.

#### help, h

Shows a list of commands or help for one command

### push

Push catalog to a remote registry

>kubara catalog push [--from oci://registry-source/repository:x.y.z] [--insecure] oci://registry-target/repository:x.y.z

**--from**="": Push an existing cached catalog reference to another registry.

**--help, -h**: show help

**--insecure**: Ignore TLS certificate verification issues for registry connections.

#### help, h

Shows a list of commands or help for one command

### login

Log into a registry and store credentials

>kubara catalog login [flags] <registry>

**--help, -h**: show help

**--identity-token**: Log in with identity token interactively

**--identity-token-stdin**: Log in with identity token from stdin

**--insecure**: Ignore TLS certificate verification issues for registry connections.

**--password, -p**: Log in with password interactively

**--password-stdin**: Log in with password from stdin

**--username, -u**="": Log in using username and password

#### help, h

Shows a list of commands or help for one command

### list, ls

List cached local and OCI-backed catalogs

>kubara catalog list

**--help, -h**: show help

#### help, h

Shows a list of commands or help for one command

### package, pkg

Package the current catalog directory into the local cache

>kubara catalog package [oci://REGISTRY/BASE/PATH/]

**--help, -h**: show help

#### help, h

Shows a list of commands or help for one command

### unpackage, unpkg

Materialize a cached OCI catalog as an editable directory

>kubara catalog unpackage oci://registry/repository:x.y.z [directory]

**--help, -h**: show help

#### help, h

Shows a list of commands or help for one command

### help, h

Shows a list of commands or help for one command

## cluster

Manage your kubara cluster configurations

>kubara cluster [command]

**--help, -h**: show help

### list, ls

List all clusters in the config file

>kubara cluster ls

**--help, -h**: show help

#### help, h

Shows a list of commands or help for one command

### add

Add a new spoke cluster to your config

>kubara cluster add CLUSTER_NAME

**--help, -h**: show help

#### help, h

Shows a list of commands or help for one command

### help, h

Shows a list of commands or help for one command

## help, h

Shows a list of commands or help for one command
