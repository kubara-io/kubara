# How to create a Catalog

This page explains how to create a custom catalog for your multi cluster platform with the `kubara catalog` command group.

If you want to understand the concepts behind catalogs and how they work in detail, please check the [Catalog concepts page](../1_getting_started/catalogs.md).

## When you need a custom catalog at all

You usually need a custom catalog when you want to:

- add a service in need on multiple clusters which kubara does not ship
- replace a built-in chart with an internal one
- change service defaults across all generated clusters
- ship reusable platform templates outside the kubara source tree

If you only want to customize values for a generated cluster, editing files in `customer-service-catalog/` is usually enough. A custom catalog is the right tool when you want to change the **service model** or the **templates kubara renders**.


## Create the catalog

Start by scaffolding the catalog root:

```bash
kubara catalog create my-catalog
```

This creates a new directory named after the catalog. The name must follow RFC 1123 naming rules: lowercase letters, digits, and `-`, starting with a letter and ending with a letter or digit.

The generated layout is:

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

The created `Catalog.yaml` looks like this:

```yaml
apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: my-catalog
```

## Add a service

Change into the catalog root and add a service definition:

```bash
cd my-catalog
kubara catalog add widget-dashboard
```

This command requires `Catalog.yaml` to be present in the current directory and creates `services/widget-dashboard.yaml`.

The generated service definition looks like this:

```yaml
apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: widget-dashboard
spec:
  chartPath: widget-dashboard
  status: disabled
  clusterTypes:
    - hub
    - spoke
```

## Continue with kubara commands

After creating the catalog and its services, use the catalog by passing the catalog root to kubara commands:

```bash
kubara schema --catalog ./my-catalog
kubara init --catalog ./my-catalog
kubara generate --catalog ./my-catalog
```

## Extending the catalog

For a **new** service that does not exist in the built-in catalog, you normally still need both:

- a `ServiceDefinition`
- matching template content under `managed-service-catalog/` and/or `customer-service-catalog/`

If you only create the `ServiceDefinition`, kubara can understand the service metadata, but it still needs actual templates to render useful output for that service.

## Overriding a built-in service

You can also override built-in services by reusing the same `metadata.name`.

Typical reasons:

- change the default `status`
- change the `chartPath`
- provide a different `configSchema`
- replace the built-in chart/templates with your own

Without `--catalog-overwrite`, kubara rejects the collision. With `--catalog-overwrite`, your external definition replaces the built-in one for that service name.

## Practical guidance

- Point `--catalog` at the **catalog root**.
- Use `kubara catalog create` and `kubara catalog add` as the entrypoint for catalog work.
- Keep `metadata.name` stable and canonical.
- Keep `chartPath` aligned with the chart directory name under `managed-service-catalog/helm/`.
- Use `configSchema` for defaults and validation instead of documenting required values only in prose.
- Treat generated files in your repo as output; treat the external catalog as the maintainable source.
