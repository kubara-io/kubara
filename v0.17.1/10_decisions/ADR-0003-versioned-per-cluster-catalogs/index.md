| status       | date       | decision-makers | consulted          | informed           |
|--------------|------------|-----------------|--------------------|--------------------|
| **accepted** | 2026-07-21 | kubara-Team     | Internal community | Internal community |

# Versioned Per-Cluster Catalog

## Context and Problem Statement

ADR-0002 established declarative `ServiceDefinition` documents and an embedded catalog with one optional external catalog. That made services extensible, but catalog selection remained global:

- The default platform catalog was compiled/embedded into the CLI
- Every cluster in one configuration saw the same merged service set
- Adding more than one external catalog was not supported
- Bootstrap, schema generation, validation, and rendering could resolve different catalog inputs
- Distributing an updated default catalog required a new CLI release

Kubara needs versioned catalogs that can evolve independently from the binary while preserving deterministic generation and strict per-cluster validation.

This ADR extends ADR-0002. It retains the `ServiceDefinition` contract and replaces only its embedded and single external catalog loading model.

## Decision Drivers

- Different clusters may require different platform stacks
- Default catalog content should be versioned independently from the CLI
- Bootstrap needs a small, stable foundation separate from configurable platform services
- Service defaults, schema generation, validation, bootstrap, and rendering must use the same catalog order
- Catalog composition must remain deterministic and explicit

## Considered Options

- Keep one embedded catalog plus one optional external catalog
- Resolve one global ordered catalog list for the entire configuration
- Resolve a bootstrap catalog plus ordered catalogs per cluster

## Decision Outcome

Chosen option: **resolve a bootstrap catalog plus ordered catalogs per cluster**.

Kubara uses:

1. A global bootstrap catalog, configured through `bootstrapCatalog` or the versioned default
2. The target cluster's ordered `catalogs` list
3. Repeated `--catalog` values appended in command-line order

Duplicate references are removed while preserving the first occurrence. Local references resolve relative to `--work-dir`. OCI references resolve through kubara's local cache and are pulled when missing.

New clusters use the versioned general catalog unless catalogs are supplied during `init` or `cluster add`. Those commands persist their catalog references. Other commands use `--catalog` as temporary additions and do not rewrite configuration.

### Bootstrap boundary

The bootstrap catalog contains `argo-cd` and `bootstrap-crds`. These services establish the GitOps foundation and are always available, but excluded from `clusters[].services`, generated service defaults, and user-facing service schemas.

The `argocd.selfManaged` field controls whether a cluster manages its own bootstrap Argo CD installation. 

### Collision and precedence semantics

Catalog order is significant:

- Service and template collisions fail by default
- Using `--catalog-overwrite` permits a later catalog to replace an earlier item
- Replacement is complete, not a deep merge
- Catalog files and selected output are processed deterministically

When multiple clusters generate the same shared output path, identical content is emitted once. Conflicting content causes generation to fail before existing output is cleaned or rewritten.

### Schema and validation

When a configuration exists, `kubara schema` resolves catalogs per cluster and emits cluster-specific service schema branches. Runtime defaulting and validation use the same per-cluster resolution.

Without a configuration file, schema generation uses the general catalog plus explicit CLI catalogs. An invalid existing configuration remains an error and does not trigger fallback behavior.

### Terraform-disabled clusters

The default `kubara generate` command still copies non-Terraform catalog assets, but skips Terraform templates for clusters with no Terraform configuration or `terraform.provider: none`. Stale generated Terraform output is removed according to the requested generation scope.

### Consequences

- **Good**, because clusters can select different catalog compositions in one configuration
- **Good**, because default catalogs can be released independently from the CLI
- **Good**, because schema, validation, bootstrap, and rendering share per-cluster resolution
- **Good**, because bootstrap-only services have an explicit and narrow boundary
- **Neutral**, because catalog order becomes part of the configuration contract
- **Bad**, because commands may need registry access when an OCI catalog is not cached
- **Bad**, because conflicting shared output across clusters is now a configuration error that users must resolve

## Non-Goals

- Deep-merging conflicting service definitions or templates
- Automatically selecting compatible catalog versions
- Signing, policy enforcement, or other catalog supply-chain controls
- A general executable plugin system
