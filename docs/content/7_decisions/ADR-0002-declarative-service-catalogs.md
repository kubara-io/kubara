| status       | date       | decision-makers  | consulted          | informed           |
|--------------|------------|------------------|--------------------|--------------------|
| **accepted** | 2026-04-23 | kubara-Team      | Internal community | Internal community |

# Declarative Service Catalogs With ServiceDefinition

## Context and Problem Statement

Kubara's service model was compile-time coupled to a fixed set of built-in services. Adding one service required edits across multiple packages and templates: a new typed field in the Go config model, hardcoded defaults in cluster factory code, hardcoded application and label entries in built-in Helm values, and the actual chart assets. Changes to core Go types require a full rebuild and new release of the CLI.

## Decision Drivers

* Kubara core had to know every service at compile time.
* Service ownership was spread across unrelated files and packages.
* External catalogs could not extend kubara without recompilation and source changes.

## Considered Options

* Keep the hardcoded registry
* Build a richer plugin framework
* Model services as declarative catalog entries via a dynamic `ServiceDefinition` type 

## Decision Outcome

Chosen option: **"Model services as declarative catalog entries via `ServiceDefinition`"**, because it fixes the root problem (compile-time service awareness) while keeping the design minimal. No complex plugin API, no service-specific Go structs in the core, no new/unique kubara-only schema language, no merge semantics between built-in and external definitions.

Each service is defined by a standalone `ServiceDefinition` YAML document. Kubara loads built-in definitions from the embedded built-in catalog and may load one optional external catalog via `--catalog`. The core operates on a generic `services` map keyed by service ID and drives validation, defaulting, schema generation, and templating from the loaded catalog. This is will therefore be a runtime extension API contract, not a compile-time "registry" anymore.

OpenAPI is used as the source format for service-specific config, inspired by [Kubernetes CustomResourceDefinitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/). Reusing `JSONSchemaProps` and Kubernetes structural defaulting is the simplest path without inventing a new DSL or API contract and not needing to implement a solved problem from scratch.

### Consequences

* **Good**, because adding a service becomes additive: definition plus chart/template assets.
* **Good**, because kubara core becomes smaller and more generic.
* **Good**, because schema generation and validation stay aligned with the same source data.
* **Good**, because built-in and external services follow the same extension path.
* **Good**, because service ownership becomes local to the catalog entry and its assets.
* **Neutral**, because the runtime contract now matters more than the Go type system did before.
* **Bad**, because some failures move from compile time to generation (run)time.

### Confirmation

A service can be added by creating a `ServiceDefinition` YAML and chart assets without any changes to kubara's core Go code and without a new CLI release. The built-in catalog itself serves as a reference: built-ins are loaded through the same catalog mechanism as external definitions and receive no special fast path inside the core.

---

## Pros and Cons of other Options

### Keep the hardcoded registry

Continue adding service-specific Go types, default logic, and template entries for every new service.

* **Good**, because failures are caught at compile time.
* **Bad**, because kubara core remains the registry of all known services.
* **Bad**, because any extension or new component requires a new build and release of the core CLI.

### Build a richer plugin framework

Introduce a general executable plugin model that services can implement.

* **Good**, because it could support arbitrary runtime extension.
* **Bad**, because it is unnecessary for the current goal. Catalogs need a strict, data-driven extension contract, not a general executable plugin model.
* **Bad**, because it significantly increases complexity without solving the immediate problem.

---

## More Information

### Decision Details

#### 1. Catalog loading model

Kubara loads:

- one embedded built-in catalog
- zero or one external catalog root passed through `--catalog`

Definitions are discovered from the catalog filesystem, collected, and processed in sorted path order.

Collision handling is explicit:

- if an external service ID collides with a built-in service ID, load fails by default
- `--catalog-overwrite` is required to allow the external definition to replace the built-in one
- replacement is whole-definition replacement, not merge

#### 2. ServiceDefinition is the source of truth

The format is:

```yaml
apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: cert-manager
  annotations:
    kubara.io/category: security
spec:
  chartPath: cert-manager
  appName: cert-manager
  status: enabled
  clusterTypes:
    - controlplane
  configSchema:
    type: object
    properties:
      clusterIssuer:
        type: object
        default: {}
        properties:
          name:
            type: string
            default: letsencrypt-staging
```

Fields:

- `apiVersion`: must be `kubara.io/v1alpha1`
- `kind`: must be `ServiceDefinition`
- `metadata.name`: name of the service
- `metadata.annotations`: optional metadata only
- `spec.chartPath`: required managed catalog chart path
- `spec.appName`: optional application name override
- `spec.status`: required default status, `enabled` or `disabled`
- `spec.clusterTypes`: optional cluster type constraint metadata
- `spec.configSchema`: optional service-specific config schema

Canonical service IDs are kebab-case names like `cert-manager` and `external-dns`.

#### 3. Schema semantics

`spec.configSchema` follows Kubernetes CRD-style `openAPIV3Schema` semantics as represented by `k8s.io/apiextensions-apiserver` `JSONSchemaProps`.

Kubara's behavior around that schema is intentionally narrow:

- stores the definition as `JSONSchemaProps`
- uses Kubernetes structural schema defaulting to apply config defaults
- converts the same schema to JSON Schema for validation and schema output
- does not define a second schema DSL on top of OpenAPI

In practice: write schemas the same way you'd write a Kubernetes CRD `openAPIV3Schema`. Kubara-specific behavior is just loading, defaulting, validating, and rendering them.

#### 4. Service instance contract in config and templates

Kubara service configuration is a generic map of service IDs to service instances.

Each service instance has a stable shape:

```yaml
services:
  cert-manager:
    status: enabled
    storage:
      className: standard-rwo
    networking:
      annotations:
        cert-manager.io/cluster-issuer: letsencrypt-staging
    config:
      clusterIssuer:
        name: letsencrypt-staging
        email: admin@example.com
        server: https://acme-staging-v02.api.letsencrypt.org/directory
```

The service instance contract is:

- `status`: core-owned desired state
- `storage.className`: core-owned storage override surface
- `networking.annotations`: core-owned ingress/network annotation override surface
- `config`: service-specific values described by `spec.configSchema` in the respective `ServiceDefinition`

`storage` and `networking` are stable kubara extension points that exist for every service instance. `config` is the schema-driven per-service area.

#### 5. Defaulting and validation order

Kubara applies defaults and validation in this order:

1. Load raw config YAML into the typed kubara config model.
2. Apply kubara's existing non-service defaults.
3. Load the effective catalog.
4. For every catalog service, ensure a service entry exists and set `status` from `spec.status` when omitted.
5. Apply `spec.configSchema` defaults using Kubernetes structural schema defaulting.
6. Generate the full config JSON Schema, including per-service schemas derived from the loaded catalog.
7. Validate the final config against that generated schema.

A few things to note:

- defaults are applied before validation
- nested config defaults work only when the parent object has its own default object (can be empty but needs to exist)
- there is no deep merge between built-in and external `ServiceDefinition` documents
- user-provided values win over catalog defaults

#### 6. Determinism rules

Runtime determinism is part of the API contract.

Kubara enforces or relies on deterministic ordering in these places:

- catalog files are processed in sorted path order
- schema service properties are emitted in sorted service-name order
- generated example app/application sections iterate services in sorted order
- conflict handling is explicit and binary: fail or replace

Given the same kubara version, catalog contents, config, and template assets, kubara should always produce the same rendered output.

#### 7. Built-in catalog becomes data, not special logic

Built-in service definitions and built-in catalog assets live in the embedded built-in catalog bundle. They are loaded through the same catalog mechanism as external definitions.

This is intentional: built-ins get no special fast path in core. Any remaining hardcoded service branching is tech debt, not a feature.

### Non-Goals

This decision does not attempt to solve all catalog ecosystem concerns in one step.

Specifically out of scope for this implementation:

- multi-catalog layering beyond built-in plus one optional external catalog
- merge semantics between conflicting definitions
- a generic plugin system
- a catalog compatibility matrix or manifest format
- signed catalog distribution or other supply-chain controls

These are real concerns, but they're follow-on work for future extensions of kubara.


### Bottom Line

Kubara will treat services as catalog data, not hardcoded product code.

The minimal stable contract is:

- versioned `ServiceDefinition` documents
- one built-in catalog plus one optional external catalog
- explicit collision behavior with replace-or-fail semantics
- CRD-style OpenAPI schema for service-specific config
- generic service instances with stable `status`, `storage`, `networking`, and `config` shape
- deterministic loading, schema generation, and rendering

That is enough to unlock extensibility now without over-engineering the system.
