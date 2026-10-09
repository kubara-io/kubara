| status       | date       | decision-makers | consulted                 | informed          |
|:-------------|:-----------|:----------------|:--------------------------|:------------------|
| **proposed** | 2026-10-08 | kubara-Team     |                           |                   |

# Integrate libkubara and Declarative CRD-Based Platform Configuration

## Context and Problem Statement

Kubara generates platform manifests and bootstraps GitOps environments across Kubernetes clusters. As features expanded, the configuration and template rendering layers have grown significantly:

1. **Unversioned configuration document**: Previous the platform settings lived in an `config.yaml` file without following a proper standard or schema versioning.
2. **Duplicated validation and defaulting**: Validation logic was scattered across Go struct tags, reflection-based JSON Schema generators (`invopop/jsonschema`, `santhosh-tekuri/jsonschema/v6`) and hand-written Go validation routines in `store.go`. 
     Defaulting required custom logic in `defaults.go`. Changing or adding any configuration field required synchronized changes across multiple files to ensure struct fields, defaults, validation functions and JSON Schema definitions.
3. **Complex cross-field rules**: Over time the project started introducing cross-field rules like certain project details only when a Terraform provider was enabled or enforcing mutual exclusion between Git and OCI repositories.
4. **Maturity of the rendering engine**: `render.go` maintained grew over time without and lacked safeguards against missing variables, output size limits, and hermetic function execution.

## Decision Drivers

* **Single source of truth**: Configuration types, defaults and validation rules *MUST* be defined once in Go code and compiled into schemas, validators and documentation.
* **Eliminate redundant code**: Replace hand-written defaulting and reflection-based schema validation with declarative tooling.
* **Kubernetes alignment**: Align the platform configuration format with Kubernetes Custom Resource standards (`apiVersion`, `kind`, `metadata`, `spec`) for future proofing it. If we ever decide to built a runtime.
* **Hardened template execution**: Enforce strict missing key errors, output size limits, hermetic template functions and better catalog collision handling.
* **Modular engine architecture**: Isolate generic manifest processing, CRD validation and template tree rendering into a reusable foundation library (`libkubara`).
* **Backward compatibility**: Existing configuration files must continue to load without manual migration by users.

## Considered Options

* **Option 1: Retain internal custom validation and template engine.** Keep `defaults.go`, custom validation loops in `store.go` and internal filesystem walking in `render.go`.
* **Option 2: Generate JSON Schema via reflection and maintain custom validation and template logic.** Keep standard JSON Schema generation tools while refining internal Go validation.
* **Option 3: Adopt libkubara and controller-gen for CRD-based configuration and engine primitives.** Transform configuration into a Kubernetes Custom Resource. Generate the CRDs with `controller-gen`, allowing for CEL rules and use of `libkubara` for validation and template rendering.

## Decision Outcome

Chosen option: "**Option 3: Adopt libkubara and controller-gen for CRD-based configuration and engine primitives**", because it establishes Go structs with Kubebuilder markers as the single source of truth for configuration loading and validation. Removes over 1000 lines of messy code, hardens the template pipeline and prepares kubara for both CLI and Kubernetes operator execution.

### Architectural changes

#### 1. Configuration as a Kubernetes Custom Resource

Configuration moves from an arbitrary YAML document to a standard Kubernetes Custom Resource:

* API version: `kubara.io/v1alpha5`
* Kind: `PlatformSetup`
* Short name: `ps`
* Scope: `Cluster`

Configuration attributes now live under `spec`, while `metadata.name` identifies the platform setup.

#### 2. Declarative validation and schema generation with controller-gen

`src/internal/config/types.go` acts as the single source of truth. Types are annotated with standard Kubebuilder markers and Common Expression Language (CEL) validation rules:

* Value constraints: `@kubebuilder:validation:Enum`, `@kubebuilder:validation:MinLength`, `@kubebuilder:validation:Format=hostname|uri|email`.
* Default values: `@kubebuilder:default=...`.
* Cross-field validation rules using CEL:
  * Exactly one hub cluster: `self.clusters.size() == 0 || self.clusters.filter(c, c.type == 'hub').size() == 1`
  * Mutually exclusive repository types: `has(self.git) != has(self.oci)`
  * Conditional Terraform requirements: `self.provider == 'none' || (has(self.projectId) && has(self.kubernetesVersion) && has(self.dns))`
  * Provider-specific Kubernetes flavors: `self.provider != 'stackit' || self.kubernetesType in ['ske', 'edge']`

The CRD is compiled using `controller-gen` via `crd_gen.go` and `make generate-crd`. The resulting definition is embedded in the binary (`kubara.io_platformsetups.yaml`).

#### 3. Validation, decoding, and transition checks via libkubara

The CLI uses `libkubara/crdvalidate` and `libkubara/manifest` to process configuration files:

* At runtime, `crdvalidate.Compile` compiles the embedded CRD into an in-memory OpenAPI v3 and CEL validator.
* `validator.ValidateCreate` validates input manifests, applies schema defaults, rejects unknown fields (`crdvalidate.RejectUnknown`), and decodes the result directly into Go structs.
* `ValidatePlatformSetupTransition` validates proposed configuration against a prior Git revision, enabling pull request checks to prevent illegal field transitions before generation runs.
* `ConfigurationJSONSchema` extracts the OpenAPI v3 schema directly from the embedded CRD for editor autocomplete via `kubara schema`.

This eliminates `defaults.go`, `invopop/jsonschema`, and `santhosh-tekuri/jsonschema/v6`.

#### 4. Hardened template rendering via libkubara/template

`render.go` delegates multi-source catalog loading and file evaluation to `libkubara/template` and `libkubara/template/tree`:

* **Hermetic Sprig:** Sprig template functions run in a hermetic mode, preventing non-deterministic environment reads.
* **Strict key validation:** `libtemplate.WithMissingKeyError()` ensures execution fails if a template references an undefined variable.
* **Resource limits:** A safety limit of 16 MB per template output prevents runaway memory usage.
* **Tree abstraction:** `tree.New` manages multi-source catalog roots, path predicates, provider path normalization, and collision resolution policies (`tree.WithCollisionResolver`).

#### 5. Structured template data context

`generator.go` builds execution contexts using `libtemplate.NewData()`. Data is cleanly namespaced into `cluster`, `env`, `catalog`, and `spokes`. This prevents naming collisions between cluster configurations and environment variables.

#### 6. Transparent legacy configuration migration

`ConfigStore.Load` detects whether a loaded file uses the legacy raw format or the `PlatformSetup` CR format. Legacy files are parsed, supplied with backward-compatible defaults via `migrate.go`, and saved as `PlatformSetup` documents without breaking user workflows.

### Consequences

* **Good**, because Go structs and Kubebuilder tags serve as the single source of truth for validation, defaulting, and schema generation.
* **Good**, because over 1000 lines of defaulting code, reflection schema generators, and directory traversal logic were deleted.
* **Good**, because CEL rules enforce cross-field dependencies declaratively without custom Go validation branches.
* **Good**, because the configuration format aligns with Kubernetes standards, unlocking transition validation in CI/CD and laying the foundation for in-cluster operator execution.
* **Good**, because template execution is protected by hermetic function execution, missing key checks, memory caps, and deterministic collision policies.
* **Good**, because legacy configurations are automatically upgraded in memory and persisted in the new format.
* **Neutral**, because developers must run `make generate-crd` whenever configuration structs change.
* **Neutral**, because automation scripts updating `config.yaml` must reference paths under `.spec`.
* **Bad**, because the CLI introduces an external dependency on `github.com/kubara-io/libkubara` that must be versioned and maintained across repositories.
