# Catalogs

This page explains what a catalog is in kubara, how catalog loading works, and how to create your own catalog manually.

For the list of services kubara ships out of the box, see the [Components Overview](../3_components/components_overview.md). That section documents the **default built-in catalog**, while this page explains the underlying catalog mechanism and how to extend it.

## What is a catalog?

In kubara, a **catalog** is the source of truth for the platform services kubara knows about.

A catalog has two responsibilities:

1. It defines **service metadata** via `ServiceDefinition` files.
2. It can provide the **template files** kubara renders into your repository.

That is different from the generated directories in your repo:

- **`managed-service-catalog/`** is generated output containing reusable Helm/Terraform artifacts.
- **`customer-service-catalog/`** is generated output containing cluster-specific overlays and values.

Those generated directories are the result of `kubara generate`. The **catalog** is the input kubara loads before it generates anything.

## Built-in vs external catalogs

kubara always ships with a **built-in catalog** embedded in the binary.

That built-in catalog includes:

- service definitions under `src/internal/catalog/built-in/services/`
- template content under `src/internal/catalog/built-in/managed-service-catalog/`

The docs section [Components (Built-in Catalog)](../3_components/components_overview.md) describes the services from that shipped default catalog.

You can extend or override that catalog by passing an **external catalog** with `--catalog`.

## How catalog loading works

When kubara loads a catalog, it does the following:

1. Loads the built-in service definitions.
2. If `--catalog` is set, loads additional service definitions from your external catalog.
3. Merges both sets by `metadata.name`.
4. Rejects name collisions unless `--catalog-overwrite` is set.

An external catalog can be structured in either of these ways for service definitions:

- `<catalog-root>/services/*.yaml`
- `<catalog-root>/*.yaml`

However, if your catalog also contains custom templates, **pass the catalog root to `--catalog`, not just the `services/` directory**. Otherwise kubara can load the `ServiceDefinition` files, but it will not find your `managed-service-catalog/` or `customer-service-catalog/` template roots.

## What a `ServiceDefinition` controls

Each service in a catalog is defined by a YAML document with:

- `apiVersion: kubara.io/v1alpha1`
- `kind: ServiceDefinition`
- `metadata.name`: canonical service name
- `spec.chartPath`: chart directory used by generated templates
- `spec.status`: default service status (`enabled` or `disabled`)
- optional `spec.clusterTypes`: limit the service to `hub` and/or `spoke`
- optional `spec.configSchema`: OpenAPI schema for service-specific config

### Example

```yaml
apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: widget-dashboard
  annotations:
    kubara.io/category: application-management
spec:
  chartPath: widget-dashboard
  status: enabled
  clusterTypes:
    - hub
    - spoke
  configSchema:
    type: object
    properties:
      hostname:
        type: string
        default: widget.example.com
      replicas:
        type: integer
        default: 2
        minimum: 1
```

## What kubara does with those fields

### `metadata.name`

This is the canonical service key used in:

- `config.yaml`
- generated schema output
- catalog merge/overwrite logic

Use **kebab-case canonical names** for new services.

### `spec.status`

This is the **default** state kubara uses when creating service entries for a cluster. Users can still override it in `config.yaml`.

### `spec.clusterTypes`

If this field is set, kubara only includes the service by default for matching cluster types:

- `hub`
- `spoke`

### `spec.configSchema`

This schema is used for service-specific configuration handling.

In practice, it affects:

- generated config schema output
- default values applied to `config.yaml`
- validation of nested `services.<name>.config` fields

For understanding available field types and the OpenAPI v3.0 structure used for `configSchema`. Please read the [Kubernetes docs](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#specifying-a-structural-schema) on how to specify a structural schema. kubara uses the upstream [kubernetes/apiextensions-apiserver](https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/apis/apiextensions/types.go) implementation for this feature and therefore fully supports all fields used for creating `CustomResourceDefinitions` as part of Kubernetes Operators.

### `spec.chartPath`

This is the directory name kubara exposes to templates as the service's chart path.

For a Helm-based service, `chartPath: widget-dashboard` usually means the chart lives at:

```text
managed-service-catalog/helm/widget-dashboard/
```

## How template loading works

If your external catalog contains template roots, kubara loads them in addition to the built-in templates.

Supported external roots are:

- `managed-service-catalog/`
- `customer-service-catalog/`

The `services/` directory is only for service definitions. It is **not** part of the rendered output.

During `kubara generate`, kubara renders templates and writes them into your repository, replacing:

- `managed-service-catalog/` with the configured managed catalog output path
- `customer-service-catalog/` with the configured customer overlay output path
- `example` path segments with the current cluster name

## Creating your own catalog manually

If you want to understand how to create a custom catalog for your multi cluster platform have a look at [How to create your own catalog](../2_managing_your_platform/create_catalog.md)
