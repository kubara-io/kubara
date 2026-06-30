# Catalog templating

This page explains how kubara renders catalog templates and allows for cross templating between helm charts, scripts, terraform
and more. 

## What is a `.tplt` file?

A file ending in `.tplt` is rendered as a Go template during `kubara generate`.
Utilizing the power of [Sprig template functions](https://masterminds.github.io/sprig/). If you authored Helm charts
before you will feel right at home.

Example:

```text
customer-service-catalog/helm/example/homer-dashboard/values.yaml.tplt
```

becomes a generated file such as:

```text
customer-service-catalog/helm/my-hub-dev/homer-dashboard/values.yaml
```

Files without the `.tplt` suffix are copied as-is.

## What data is available in templates?

kubara builds a template context with three top-level objects:

- `.cluster`
- `.env`
- `.catalog`

### `.cluster`

`.cluster` contains the current cluster entry from `config.yaml`.

That means templates can read fields such as:

- `.cluster.name`
- `.cluster.stage`
- `.cluster.type`
- `.cluster.dnsName`
- `.cluster.ingressClassName`
- `.cluster.publicLoadbalancerIP`
- `.cluster.terraform.provider`

It also includes the current cluster's resolved services under:

```text
.cluster.services.<service-name>
```

For example:

- `.cluster.services.traefik.status`
- `.cluster.services.cert-manager.config.clusterIssuer.name`

### `.env`

`.env` exposes the loaded `.env` values that kubara already uses while running.

Use this only when a template really needs environment-derived data.

### `.catalog`

`.catalog` exposes catalog metadata for all known services.

Today this is mainly service metadata such as:

- `.catalog.services.<service-name>.status`
- `.catalog.services.<service-name>.chartPath`
- `.catalog.services.<service-name>.clusterTypes`

This is useful when template logic needs catalog-level defaults or service metadata.

## Cross-templating in practice

The Homer example shows the main idea well: one service template can react to settings from other services and from the cluster itself.

From:

```text
src/internal/catalog/built-in/customer-service-catalog/helm/example/homer-dashboard/values.yaml.tplt
```

kubara reads values such as:

- The current cluster DNS name `.cluster.dnsName`
- Whether `oauth2-proxy` is enabled
- Whether `traefik` is enabled
- Whether `metallb` is enabled
- Values from the `cert-manager` service config

This is called **cross-templating**: A template for one service can use data from:

- The cluster
- Other services in the same cluster
- Catalog metadata

## Small example

```yaml
# Dig can traverse a nested map like "cluster.services.<service-name>.status" while staying null safe
# Meaning if any of the keys listed below is missing it will default to the last value in the
# list below: "disabled"
{{- $oauth2ProxyStatus := dig "cluster" "services" "oauth2-proxy" "status" "disabled" . -}}
{{- $traefikStatus := dig "cluster" "services" "traefik" "status" "disabled" . -}}

ingress:
  {{- if (eq $oauth2ProxyStatus "enabled") }}
  enabled: true
  {{- end }}
  host: {{ .cluster.dnsName }}
```

kubara templates can use the [Sprig template functions](https://masterminds.github.io/sprig/).
The [`dig` function](https://masterminds.github.io/sprig/dicts.html#dig) is especially useful when you need to read nested fields that might be missing or `null`.

This snippet does two things:

1. It checks service state from `.cluster.services`
2. It renders the hostname from `.cluster.dnsName`

That is the core power of kubara templating: One template can adapt itself to the full cluster configuration and it can
be applied independent of the ecosystem no matter if the resulting file will be HCL for Terraform, yaml templates for Helm
charts, custom scripts or others.

## Good template authoring rules

- Prefer simple conditions and simple defaults.
- Read from `.cluster` first when the value is cluster-specific.
- Use service config instead of hard-coding environment-specific values.
- Keep cross-service dependencies obvious and at the top of your templates.
- Keep chart structure and template path names predictable.

## Terraform-specific note

Provider-specific Terraform template variants are supported below:

```text
terraform/providers/<provider>/
```

kubara strips that provider path segment in the final output and picks the provider-specific file when the cluster provider matches.

## Where to go next

- To understand the catalog model: [Catalogs](../2_concepts/catalogs.md)
- To package and share catalogs: [Catalog distribution](../2_concepts/catalog_distribution.md)
- To author a full catalog: [How to create a Catalog](../4_building_your_platform/create_catalog.md)
