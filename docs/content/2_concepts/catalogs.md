# Catalogs

This page explains what a catalog is, when to use one, and how catalogs fit into kubara's platform workflow.

For the built-in services that ship with kubara, see [Components Overview](../5_components/components_overview.md). This page is about the **catalog model itself**.

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

- [Add a Project](../4_workload_onboarding/add_app_project.md)
- [Add a Repository](../4_workload_onboarding/add_app_repository.md)
- [Add an ApplicationSet](../4_workload_onboarding/add_appset.md)
- [Add an Application](../4_workload_onboarding/add_application.md)

Use the Argo CD guides in [Workload Onboarding with Argo CD](../4_workload_onboarding/overview.md) for that path.

Hint:

- Use Argo CD workload onboarding when the service you are adding is a **cluster-specific or team-specific workload**.
- Use a catalog only when the service you are describing is part of the **reusable platform architecture**.  

## Built-in and external catalogs

kubara always ships with a **built-in catalog**:

That built-in catalog contains:

- built-in service definitions
- built-in template content
- the default platform stack documented in the Components section

You can extend or replace parts of it with an **external catalog** by using `--catalog`.

Examples:

```bash
kubara schema --catalog oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
kubara init --catalog oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
kubara generate --catalog oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
```

`--catalog` accepts either:

- a local catalog directory
- an OCI reference such as `oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3`

Important: OCI-backed catalogs are resolved from the **local kubara cache**. If the catalog is not cached yet, pull it first with `kubara catalog pull`. See [Catalog distribution](catalog_distribution.md).

## What is inside a catalog?

A catalog usually has these parts:

```text
my-catalog/
├── Catalog.yaml
├── services/
├── managed-service-catalog/
│   ├── helm/
│   └── terraform/
└── customer-service-catalog/
    ├── helm/
    │   └── example/
    └── terraform/
        └── example/
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

### `managed-service-catalog/`

This contains reusable generated output sources, for example:

- Helm charts
- Terraform modules
- shared assets

### `customer-service-catalog/`

This contains cluster-specific overlays and values templates.

The `example/` path segment is special. During `kubara generate`, kubara replaces it with the current cluster name.

For example:

```text
customer-service-catalog/helm/example/homer-dashboard/values.yaml.tplt
```

becomes:

```text
customer-service-catalog/helm/<cluster-name>/homer-dashboard/values.yaml
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

Without `--catalog-overwrite`, kubara rejects collisions between built-in and external service definitions with the same name.

## How catalog loading works

When kubara loads catalogs, it:

1. Loads the built-in service definitions
2. Loads external service definitions when `--catalog` is set
3. Merges both sets by `metadata.name`
4. Rejects collisions unless `--catalog-overwrite` is set

## How template loading works

During `kubara generate`, kubara loads templates from:

- The built-in catalog
- Your external catalog, if present

Files ending in `.tplt` are rendered as Go templates. Files without `.tplt` are copied as-is.

For Terraform, kubara also supports provider-specific template variants below:

```text
terraform/providers/<provider>/
```

If a provider-specific file and a common file map to the same output path, the provider-specific file wins.

If a cluster has no Terraform block or uses `terraform.provider: none`, the default `kubara generate` run skips Terraform templates for that cluster.

## OCI-backed distribution

Catalogs can be packaged and distributed as OCI artifacts.

For the full workflow, see [Catalog distribution](catalog_distribution.md).

OCI is the same ecosystem standard used by container images, Helm registries, and many other Kubernetes tools. kubara uses OCI so catalogs can move through the same registry infrastructure that many teams already use.

Read more:

- [ORAS: OCI artifacts](https://oras.land/docs/concepts/artifact)
- [ORAS: reference types](https://oras.land/docs/concepts/reftypes)


## Where to go next

- To build your own catalog: [How to create a Catalog](../3_building_your_platform/create_catalog.md)
- To distribute catalogs through a registry: [Catalog distribution](catalog_distribution.md)
- To learn template authoring: [Catalog templating](catalog_templating.md)
- To add simpler workloads through Argo CD instead: [Workload Onboarding with Argo CD](../4_workload_onboarding/overview.md)
