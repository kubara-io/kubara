# Catalogs

This page explains what a catalog is, when to use one, and how catalogs fit into kubara's platform workflow.

For the general services the kubara team ships, see [Components Overview](../6_components/components_overview.md). This page is about the **catalog model itself**.

## What is a catalog?

In kubara, a **catalog** is a packaged and templateable platform setup.

It is the input that kubara uses to generate your platform artifacts. A catalog can define:

- Service Metadata
- Helm charts
- Terraform modules
- Kustomize files
- Scripts
- Any other files your platform setup needs

In a nutshell: If Helm charts are packages for a single application, **kubara catalogs are packages for your platform architecture**.

## Why do catalogs exist?

Catalogs are useful when you need to roll out the **same platform design** again and again across many clusters.

Typical examples:

- A company platform that must be deployed in many regions
- A partner platform that is reused for many customers
- An internal platform baseline for dozens of clusters
- A platform stack that combines infrastructure, GitOps, security, and observability in one repeatable unit

The main idea is simple:

1. Define the reusable platform setup once.
2. Store cluster-specific intent in `config.yaml`.
3. Run `kubara generate`.
4. Let kubara render the final Terraform and Helm output for each cluster.

## When **not** to use a catalog

Do **not** create a catalog service for every workload.

If a workload belongs to one cluster, one team, or one application domain, it is usually simpler to add it through Argo CD:

- [Add a Project](../5_workload_onboarding/add_app_project.md)
- [Add a Repository](../5_workload_onboarding/add_app_repository.md)
- [Add an ApplicationSet](../5_workload_onboarding/add_appset.md)
- [Add an Application](../5_workload_onboarding/add_application.md)

Use the Argo CD guides in [Workload Onboarding with Argo CD](../5_workload_onboarding/overview.md) for that path.

Hint:

- Use Argo CD workload onboarding when the service you are adding is a **cluster-specific or team-specific workload**.
- Use a catalog only when the service you are describing is part of the **reusable platform architecture**.  

## Bootstrap and cluster catalogs

kubara resolves catalogs in layers:

1. The global **bootstrap catalog** provides the fixed Argo CD and bootstrap CRD foundation.
2. Each cluster selects one or more ordered catalogs through its `catalogs` list.
3. Repeated `--catalog` flags append command-specific catalogs after the cluster catalogs.

New configurations use kubara's general catalog unless catalogs are supplied during `init`. The selected catalog references are stored on the cluster:

```yaml
bootstrapCatalog: oci://ghcr.io/kubara-io/catalogs/bootstrap:1.0.0
clusters:
  - name: production
    catalogs:
      - oci://ghcr.io/kubara-io/catalogs/general:1.0.0
      - oci://ghcr.io/acme/platform-catalogs/security:2.1.0
```

`bootstrapCatalog` is optional. When omitted or empty, kubara uses its versioned default bootstrap catalog. The bootstrap services `argo-cd` and `bootstrap-crds` are always part of the foundation and are not configurable entries under `clusters[].services`.

Use `--catalog` to add catalogs for a command:

Examples:

```bash
kubara schema --catalog oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
kubara init --catalog oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
kubara generate --catalog oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
```

`--catalog` accepts either:

- a local catalog directory
- an OCI reference such as `oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3`

Local paths are resolved relative to `--work-dir`. OCI-backed catalogs use the local kubara cache and are pulled automatically when the requested reference is not cached. See [Catalog distribution](catalog_distribution.md).

`init` and `cluster add` persist their `--catalog` references in the new cluster entry. Commands such as `schema`, `generate`, and `bootstrap` use CLI catalogs as temporary additions and do not rewrite `config.yaml`.

## What is inside a catalog?

A catalog usually has these parts:

```text
my-catalog/
├── Catalog.yaml
├── services/
├── platform-components/
│   ├── helm/
│   └── terraform/
└── platform-configs/
    ├── helm/
    └── terraform/
```

### `Catalog.yaml`

This is the catalog manifest. It identifies the catalog and its version.

Example:

```yaml
apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: my-catalog
spec:
  version: 1.2.3
```

The version must be a plain `major.minor.patch` version.

### `services/`

This directory contains `ServiceDefinition` files.

Each file tells kubara things like:

- The canonical service name
    - Following kubernetes conventions
    - RFC1123: No upper letters, underscores or special characters besides dashes
- The default deployment status
- The chart path inside the catalog
- Optional cluster type limits
    - hub & spoke or only one of the two
- Optional service config schema

### `platform-components/`

This contains reusable generated output sources, for example:

- Helm charts
- Terraform modules
- shared assets

### `platform-configs/`

This contains cluster-specific overlays and values templates.

For example:

```text
platform-configs/helm/homer-dashboard/values.generated.yaml.tplt
```

becomes:

```text
platform-configs/<cluster-name>/helm/homer-dashboard/values.generated.yaml
```

## What a `ServiceDefinition` controls

Each service definition is a YAML document like this:

```yaml
apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: pet-store
spec:
  chartPath: pet-store
  status: enabled
  clusterTypes:
    - hub
    - spoke
  configSchema:
    type: object
    properties:
      hostname:
        type: string
        default: pets.example.com
```

Important fields:

- `metadata.name`: the canonical service key used in `config.yaml`
- `spec.chartPath`: the chart directory name used by templates
- `spec.status`: default service status for new clusters
- `spec.clusterTypes`: optional hub/spoke filtering
- `spec.configSchema`: optional OpenAPI schema for defaults and validation

Without `--catalog-overwrite`, kubara rejects collisions between service definitions with the same name.

## How catalog loading works

For each cluster, kubara:

1. Loads the bootstrap catalog.
2. Loads `clusters[].catalogs` in their listed order.
3. Appends repeated `--catalog` values in command-line order.
4. Removes duplicate references while preserving the first occurrence.
5. Merges service definitions by `metadata.name`.
6. Rejects collisions unless `--catalog-overwrite` is set.

With overwrite enabled, the later catalog replaces the complete earlier definition; definitions are not deep-merged.

## How template loading works

During `kubara generate`, kubara loads templates from the same effective catalog order used for service defaults and validation.

Files ending in `.tplt` are rendered as Go templates. Files without `.tplt` are copied as-is.

For Terraform, kubara also supports provider-specific template variants below:

```text
platform-configs/terraform/<provider>/
platform-components/terraform/<provider>/
```

If a provider-specific file and a common file map to the same output path, the provider-specific file wins. The selector is removed from generated `platform-configs` paths and retained in `platform-components` module paths.

If a cluster has no Terraform block or uses `terraform.provider: none`, the default `kubara generate` run skips Terraform templates for that cluster.

Shared output paths must also be deterministic across clusters. Identical content is written once; conflicting content for the same final path causes generation to fail before files are changed.

## Schema generation

When `config.yaml` exists, `kubara schema` resolves catalogs per cluster and emits cluster-specific service branches. This keeps editor completion and validation aligned with the services available to each cluster.

Without a configuration file, `kubara schema` uses the general catalog plus any repeated `--catalog` values. If an existing configuration is malformed or references an invalid catalog, schema generation fails instead of silently falling back.

## OCI-backed distribution

Catalogs can be packaged and distributed as OCI artifacts.

For the full workflow, see [Catalog distribution](catalog_distribution.md).

OCI is the same ecosystem standard used by container images, Helm registries, and many other Kubernetes tools. kubara uses OCI so catalogs can move through the same registry infrastructure that many teams already use.

Read more:

- [ORAS: OCI artifacts](https://oras.land/docs/concepts/artifact)
- [ORAS: reference types](https://oras.land/docs/concepts/reftypes)


## Where to go next

- To build your own catalog: [How to create a Catalog](../4_building_your_platform/create_catalog.md)
- To distribute catalogs through a registry: [Catalog distribution](catalog_distribution.md)
- To learn template authoring: [Catalog templating](catalog_templating.md)
- To add simpler workloads through Argo CD instead: [Workload Onboarding with Argo CD](../5_workload_onboarding/overview.md)
