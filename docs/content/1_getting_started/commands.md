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

**--base64**: Enable base64 encode/decode mode (default: false)

**--catalog**="": Path to a catalog directory or an OCI reference in the form oci://registry/repository:x.y.z

**--catalog-overwrite**: Allow later catalogs from --catalog to overwrite earlier definitions on name collisions (default: false)

**--check-update**: Check online for a newer kubara release (default: false)

**--config-file, -c**="": Path to the configuration file (default: config.yaml)

**--decode**: Base64 decode input (default: false)

**--encode**: Base64 encode input (default: false)

**--env-file**="": Path to the .env file (default: .env)

**--file**="": Input file path for base64 operation

**--help, -h**: show help (default: false)

**--kubeconfig**="": Path to kubeconfig file (default: ~/.kube/config)

**--string**="": Input string for base64 operation

**--test-connection**: Check if Kubernetes cluster can be reached. List namespaces and exit (default: false)

**--version, -v**: print the version (default: false)

**--work-dir, -w**="": Working directory (default: .)


# COMMANDS

## init

Initialize kubara config for your GitOps repository

>kubara init [--prep] [--local] [--renovate=false] [--github-actions] [--gitlab-ci] [--bootstrap-catalog PATH_OR_OCI]

**--bootstrap-catalog**="": Path to the bootstrap catalog directory or an OCI reference in the form oci://registry/repository:x.y.z

**--envVarPrefix**="": Prefix for envs read from envVars (default: KUBARA_)

**--github-actions**: Generate a GitHub Actions workflow for kubara generation if none exists (default: false)

**--gitlab-ci**: Generate a GitLab CI pipeline for kubara generation if none exists (default: false)

**--help, -h**: show help (default: false)

**--local**: Initialize files for the local evaluation workflow. Local testing only; not for production use. (default: false)

**--overwrite**: Overwrite config if exists (default: false)

**--prep**: Copy embedded prep/ folder into current working directory (default: false)

**--renovate**: Generate a Renovate configuration for kubara catalog updates if none exists (default: true)

### help, h

Shows a list of commands or help for one command

## generate

Generate files from catalog templates

>kubara generate [--terraform|--helm] [--catalog PATH_OR_OCI [--catalog-overwrite]] [--dry-run]

**--dry-run**: Preview generation without creating files (default: false)

**--helm**: Only generate Helm files (default: false)

**--help, -h**: show help (default: false)

**--terraform**: Only generate Terraform files (default: false)

### help, h

Shows a list of commands or help for one command

## bootstrap

Bootstrap Argo CD onto a cluster

>kubara bootstrap CLUSTER_NAME [--local]

**--dry-run**: Run with dry-run (default: false)

**--envVarPrefix**="": Prefix for envs read from envVars (default: KUBARA_)

**--help, -h**: show help (default: false)

**--local**: Provision an isolated local evaluation environment. Local testing only; not for production use. (default: false)

**--platform-components**="": Path to the platform-components directory (default: platform-components)

**--platform-configs**="": Path to platform-configs directory (default: platform-configs)

**--timeout**="": Timeout for kubernetes API calls (e.g. 10s, 1m) (default: 5m0s)

**--with-es-crds**: Deprecated: ignored because CRDs are applied automatically during bootstrap. (default: false)

**--with-es-css-file**="": Path to the ClusterSecretStore manifest file (supports go-template + sprig)

**--with-prometheus-crds**: Deprecated: ignored because CRDs are applied automatically during bootstrap. (default: false)

### help, h

Shows a list of commands or help for one command

## schema

Generate a JSON schema for the config yaml structure

>kubara schema [--output PATH]

**--help, -h**: show help (default: false)

**--output, -o**="": Output file path for the JSON schema (default: config.schema.json)

### help, h

Shows a list of commands or help for one command

## agents

Scaffold an onboarding file for AI coding assistants (AGENTS.md)

>kubara agents [--overwrite]

**--help, -h**: show help (default: false)

**--overwrite**: Overwrite an existing AGENTS.md (default: false)

### help, h

Shows a list of commands or help for one command

## catalog

Manage platform catalogs

>kubara catalog [command]

**--help, -h**: show help (default: false)

### create

Create a custom catalog directory skeleton

>kubara catalog create [--github-actions] [--gitlab-ci] CATALOG_NAME

**--github-actions**: Include a GitHub Actions catalog validation workflow (default: false)

**--gitlab-ci**: Include a GitLab catalog validation pipeline (default: false)

**--help, -h**: show help (default: false)

#### help, h

Shows a list of commands or help for one command

### add

Add a service definition to the current catalog

>kubara catalog add SERVICE_NAME

**--help, -h**: show help (default: false)

#### help, h

Shows a list of commands or help for one command

### pull

Pull a catalog from a remote registry

>kubara catalog pull [--insecure] oci://registry/repository:x.y.z

**--help, -h**: show help (default: false)

**--insecure**: Ignore TLS certificate verification issues for the registry connection. (default: false)

#### help, h

Shows a list of commands or help for one command

### push

Push catalog to a remote registry

>kubara catalog push [--from oci://registry-source/repository:x.y.z] [--insecure] oci://registry-target/repository:x.y.z

**--from**="": Push an existing cached catalog reference to another registry.

**--help, -h**: show help (default: false)

**--insecure**: Ignore TLS certificate verification issues for registry connections. (default: false)

#### help, h

Shows a list of commands or help for one command

### login

Log into a registry and store credentials

>kubara catalog login [flags] <registry>

**--help, -h**: show help (default: false)

**--identity-token**="": identity token

**--identity-token-stdin**: Log in with identity token from stdin (default: false)

**--insecure**: Ignore TLS certificate verification issues for registry connections. (default: false)

**--password, -p**="": registry password

**--password-stdin**: Log in with password from stdin (default: false)

**--username, -u**="": registry username

#### help, h

Shows a list of commands or help for one command

### list, ls

List cached local and OCI-backed catalogs

>kubara catalog list

**--help, -h**: show help (default: false)

#### help, h

Shows a list of commands or help for one command

### package, pkg

Package the current catalog directory into the local cache

>kubara catalog package [oci://REGISTRY/BASE/PATH/]

**--help, -h**: show help (default: false)

#### help, h

Shows a list of commands or help for one command

### unpackage, unpkg

Materialize a cached OCI catalog as an editable directory

>kubara catalog unpackage oci://registry/repository:x.y.z [directory]

**--help, -h**: show help (default: false)

#### help, h

Shows a list of commands or help for one command

### help, h

Shows a list of commands or help for one command

## cluster

Manage your kubara cluster configurations

>kubara cluster [command]

**--help, -h**: show help (default: false)

### list, ls

List all clusters in the config file

>kubara cluster ls

**--help, -h**: show help (default: false)

#### help, h

Shows a list of commands or help for one command

### add

Add a new spoke cluster to your config

>kubara cluster add CLUSTER_NAME

**--help, -h**: show help (default: false)

#### help, h

Shows a list of commands or help for one command

### help, h

Shows a list of commands or help for one command

## help, h

Shows a list of commands or help for one command
