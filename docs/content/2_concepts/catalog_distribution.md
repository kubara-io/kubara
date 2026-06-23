# Catalog distribution

This page explains how to package, publish, pull, and use kubara catalogs as OCI artifacts.

## Why kubara uses OCI

[OCI (Open Container Initiative)](https://opencontainers.org/) is the artifact format and distribution model behind container images. It is also widely used in the Kubernetes ecosystem, including Helm registries and other registry-backed artifacts.

kubara uses OCI so catalogs can be:

- Versioned
- Cached locally
- Pushed to standard registries
- Pulled in other environments
- Promoted between registries

Common registries that support OCI include:

- AWS Elastic Container Registry (ECR)
- Azure Container Registry (ACR)
- GitHub Container Registry (`ghcr.io`)
- Google Artifact Registry
- Harbor
- STACKIT Container Registry

If you want the deeper OCI background, read:

- [ORAS: OCI artifacts](https://oras.land/docs/concepts/artifact)
- [ORAS: reference types](https://oras.land/docs/concepts/reftypes)

## Important model: local cache first

kubara works with a **local catalog cache**.

That means:

- `kubara catalog package` creates a cached catalog artifact from a local directory
- `kubara catalog pull` downloads a catalog artifact into that cache
- `kubara catalog push` uploads a catalog artifact that is already in that cache
- `--catalog oci://...` resolves an OCI reference from that cache

In other words:

- **push does not package**
- **generate does not auto-pull**

Package or pull first, then use the cached artifact.

## Step 1: Create or update the catalog

Start with a normal catalog directory:

```bash
kubara catalog create my-catalog
cd my-catalog
kubara catalog add pet-store
```

Edit the service definitions and template files until the catalog is ready.

## Step 2: Package the catalog into the local cache

Package the current catalog directory:

```bash
kubara catalog package oci://ghcr.io/acme/platform-catalogs/
```

kubara reads:

- `metadata.name` from `Catalog.yaml`
- `spec.version` from `Catalog.yaml`

and builds the final reference from that information.

Example `Catalog.yaml`:

```yaml
apiVersion: kubara.io/v1alpha1
kind: Catalog
metadata:
  name: my-catalog
spec:
  version: 1.2.3
```

Example result:

```text
oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
```

The packaged artifact includes the files inside the catalog directory, so your catalog can ship more than just Helm and Terraform sources.

## Step 3: Log into the registry

If your registry needs authentication, log in once:

```bash
kubara catalog login -u my-github-user --password ghcr.io
```

kubara stores registry credentials in:

```text
$HOME/.kubara/credentials.json
```

You can use:

- username/password (interactive)
- password from stdin
- identity token (interactive)
- identity token from stdin

## Step 4: Push the cached catalog

Push the cached artifact to the registry:

```bash
kubara catalog push oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
```

Important:

- The destination must use a **tag**, not a digest
- The destination tag must match `spec.version` from `Catalog.yaml`
- The artifact must already exist in the local cache

You can also promote an already cached catalog from one reference to another:

```bash
kubara catalog push \
  --from oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3 \
  oci://registry.example.com/platform/my-catalog:1.2.3
```

This is useful when you want to copy the same packaged version into another registry or repository path.

## Step 5: Pull the catalog somewhere else

On another machine or in CI, pull the catalog into the local cache:

```bash
kubara catalog pull oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
```

If the cached version already exists and the remote digest changed, kubara updates the cached entry.

## Step 6: Use the pulled catalog with kubara

After the catalog is cached locally, you can use the same OCI reference in kubara commands:

```bash
kubara schema --catalog oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
kubara init --catalog oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
kubara generate --catalog oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3
```

If the catalog is not cached yet, pull it first.

## Useful supporting commands

List cached artifacts:

```bash
kubara catalog list
```

Materialize a cached artifact as an editable directory:

```bash
kubara catalog unpackage oci://ghcr.io/acme/platform-catalogs/my-catalog:1.2.3 ./my-catalog
```

This is useful when you want to inspect or edit a catalog that was distributed through a registry.
