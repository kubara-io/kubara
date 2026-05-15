# How to create a Catalog

This page explains how to create a custom catalog for your multi cluster platform.

If you want to understand the concepts behind catalogs and how they work in detail, please check the [Catalog concepts page](../1_getting_started/catalogs.md).

## When you need a custom catalog at all

You usually need a custom catalog when you want to:

- add a service in need on multiple clusters which kubara does not ship
- replace a built-in chart with an internal one
- change service defaults across all generated clusters
- ship reusable platform templates outside the kubara source tree

If you only want to customize values for a generated cluster, editing files in `customer-service-catalog/` is usually enough. A custom catalog is the right tool when you want to change the **service model** or the **templates kubara renders**.


## Structure

The layout for a custom catalog is:

```text
my-catalog/
├── services/
│   └── widget-dashboard.yaml
├── managed-service-catalog/
│   └── helm/
│       └── widget-dashboard/
│           ├── Chart.yaml
│           ├── values.yaml
│           └── templates/
│               └── ...
└── customer-service-catalog/
    └── helm/
        └── example/
            └── widget-dashboard/
                └── values.yaml.tplt
```

## Minimal steps

1. Create a `ServiceDefinition` in `services/`.
2. Add the matching chart or templates under `managed-service-catalog/`.
3. Optionally add cluster overlay templates under `customer-service-catalog/`.
4. Run kubara commands with `--catalog /path/to/my-catalog`.

## Example commands

```bash
kubara schema --catalog ./my-catalog
kubara init --catalog ./my-catalog
kubara generate --catalog ./my-catalog
```

## Adding a completely new service

For a **new** service that does not exist in the built-in catalog, you normally need both:

- a `ServiceDefinition`
- matching template content under `managed-service-catalog/` and/or `customer-service-catalog/`

If you only add the `ServiceDefinition`, kubara can understand the service metadata, but it still needs actual templates to render useful output for that service.

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
- Keep `metadata.name` stable and canonical.
- Keep `chartPath` aligned with the chart directory name under `managed-service-catalog/helm/`.
- Use `configSchema` for defaults and validation instead of documenting required values only in prose.
- Treat generated files in your repo as output; treat the external catalog as the maintainable source.

