# How to create a Catalog

This page explains how to author a custom catalog for kubara and how to prepare it for distribution.

If you are new to the idea itself, read [Catalogs](../2_concepts/catalogs.md) first.

## When you need a custom catalog

You usually need a custom catalog when you want to:

- Add a platform service that kubara does not ship
- Replace a chart or module provided in another catalog with your own
- Change platform defaults across many clusters
- Package and distribute a reusable platform setup outside the kubara source tree

If you only need to add a simpler workload to one cluster or one team space, use the Argo CD guides in [Workload Onboarding with Argo CD](../5_workload_onboarding/overview.md) instead.

## Step 1: scaffold the catalog

Create the catalog root:

```bash
kubara catalog create my-catalog
```

The catalog name must follow RFC 1123 naming rules:

- lowercase letters
- digits
- `-`
- starts with a letter
- ends with a letter or digit

Generated structure:

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

The generated `Catalog.yaml` looks like this:

```yaml
apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: my-catalog
spec:
  version: 0.1.0
```

`spec.version` is important because kubara uses it when packaging the catalog as an OCI artifact.

kubara enforces strict semantic version formatting for catalogs:

- allowed: `0.1.0`
- not allowed: `v0.1.0`
- not allowed: `0.1.0-rc.1`
- not allowed: `0.1.0-beta`
- not allowed: `0.1.0+build.5`

Only plain `major.minor.patch` is accepted.

## Step 2: add service definitions

Move into the catalog root and add a service:

```bash
cd my-catalog
kubara catalog add pet-store
```

This creates `services/pet-store.yaml`.

Example:

```yaml
apiVersion: kubara.io/v1alpha1
kind: ServiceDefinition
metadata:
  name: pet-store
spec:
  chartPath: pet-store
  status: disabled
  clusterTypes:
    - hub
    - spoke
```

Key fields:

- `metadata.name`: stable service key used in `config.yaml`
- `spec.chartPath`: chart directory name under `platform-components/helm/`
- `spec.status`: default status
- `spec.clusterTypes`: optional hub/spoke limit
- `spec.configSchema`: optional schema for service-specific config

## Step 3: add the actual platform content

A service definition alone is not enough. kubara also needs the files it should render or copy.

Common places:

- `platform-components/helm/<chart>/`
- `platform-components/terraform/...`
- `platform-configs/helm/<chart>/`
- `platform-configs/terraform/...`

Use `platform-components/` for reusable source content.  
Use `platform-configs/` for cluster-specific overlays.

If you want to learn how `.tplt` files work, read [Catalog templating](../2_concepts/catalog_templating.md).

## Step 4: use the catalog with kubara

Point kubara at the catalog root:

```bash
kubara schema --catalog ./my-catalog
kubara init --catalog ./my-catalog
kubara generate --catalog ./my-catalog
```

Pass the **catalog root**, not only `services/`, when you also want kubara to load templates.

You can also assign catalogs directly to a cluster:

```yaml
clusters:
  - name: production
    catalogs:
      - ./my-catalog
      - oci://ghcr.io/acme/platform-catalogs/security:1.4.0
```

Catalog order is significant. Cluster catalogs are loaded first in the listed order, followed by repeated `--catalog` values. Local references are resolved relative to `--work-dir`.

`kubara schema` automatically discovers cluster catalogs when `config.yaml` exists. Before creating a configuration, pass the catalog explicitly as shown above.

## Step 5: override services when needed

You can override a service defined with the same `metadata.name` from a previous catalog.

Typical reasons:

- Change the default `status`
- Change the `chartPath`
- Provide a different `configSchema`
- Replace templates completely with your own

Without `--catalog-overwrite`, kubara rejects the collision.  
With `--catalog-overwrite`, the later catalog replaces the earlier definition for that service name. The replacement is complete rather than a deep merge.

## Step 6: package the catalog

When the catalog is ready, package it into the local cache:

```bash
kubara catalog package oci://ghcr.io/acme/platform-catalogs/
```

kubara derives the final reference from:

- `metadata.name`
- `spec.version`
- the optional OCI base path you pass

That means packaging is versioned by `Catalog.yaml`, not by the directory name alone.

## Step 7: distribute the catalog

After packaging, you can:

- log into a registry
- push the cached artifact
- pull it somewhere else
- use the pulled OCI reference with `--catalog`

See [Catalog distribution](../2_concepts/catalog_distribution.md) for the full workflow.

## Provider-specific Terraform templates

Provider-specific template variants are supported directly below a Terraform directory:

```text
platform-configs/terraform/<provider>/
platform-components/terraform/<provider>/
```

Example:

```text
platform-configs/terraform/stackit/infrastructure/main.tf.tplt
```

When the cluster Terraform provider matches the directory name, kubara selects that provider's files. The provider directory is stripped from generated `platform-configs` paths but retained under `platform-components`.

Provider-specific directories below Helm paths are **not** treated as provider overrides.

## Practical guidance

- Keep `metadata.name` stable.
- Keep `spec.version` stable until you intentionally publish a new version.
- Keep `chartPath` aligned with the actual Helm chart directory name.
- Use `configSchema` for defaults and validation instead of prose only.
- Treat the catalog directory as the maintainable source.
- Treat generated files in your repo as output.
