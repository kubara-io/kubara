# How to automate catalog updates

This page explains how to update the bootstrap and general catalogs in a GitOps repository using GitHub Actions or GitLab CI.
The pipelines run `kubara generate` and propose configuration and manifest changes in a pull request or merge request for review.

For background on OCI catalog versions, read [Catalog distribution](../2_concepts/catalog_distribution.md).

## Prerequisites

- An existing kubara configuration and generated artifacts committed to a GitOps repository
- Explicit stable `x.y.z` catalog versions in `config.yaml`
- A runner with access to the catalog registry and kubara releases
- Permission to push an automation branch and create a PR or MR

The scripts use Bash, Mike Farah yq v4, jq, ORAS, and kubara.
The GitHub example uses the hosted Ubuntu runner's yq and jq. The GitLab example installs its tools in an Alpine container.

## Step 1: prepare the configuration

Set `bootstrapCatalog` and a nonempty `catalogs` list for every cluster.
Keep your existing cluster settings and make the catalog versions explicit:

```yaml
bootstrapCatalog: oci://ghcr.io/kubara-io/catalogs/bootstrap:5.0.1
clusters:
  - name: production
    # Keep the remaining cluster settings here.
    catalogs:
      - oci://ghcr.io/kubara-io/catalogs/general:5.1.0
```

This is a configuration fragment. The versions illustrate the required format.
The updater targets the public `ghcr.io/kubara-io/catalogs/bootstrap` and `ghcr.io/kubara-io/catalogs/general` repositories.
Both must have explicit version tags; digest pins and implicit defaults are not supported by this example.

If other catalogs in your configuration are private, run `kubara catalog login --password-stdin` with CI credentials before generation.
Provide any environment variables needed by your templates through CI variables or secrets.

## Step 2: add the pipeline

Run `kubara init` with the option for your Git hosting provider.
The command generates the pipeline and its required scripts in your working directory.
Existing files are preserved, including when using `--overwrite`. The `--prep` option only prepares the environment file.

=== "GitHub Actions"

    ```bash
    kubara init --github-actions
    ```

    This creates `.github/workflows/kubara-update.yaml` and `.scripts/kubara-catalog-update.sh`.

    Enable **Allow GitHub Actions to create and approve pull requests** in the repository's Actions settings.
    The workflow requires `contents: write` and `pull-requests: write`.

    Replace `kubara-io/kubara@main` with a reviewed commit or release containing the action.
    The workflow runs daily at 04:00 UTC and can be started with **Run workflow**.

    The default `GITHUB_TOKEN` does not trigger most downstream workflows on the generated PR.
    If PR checks need to run automatically, provide an appropriately scoped GitHub App token to the checkout and PR creation steps.

=== "GitLab CI"

    ```bash
    kubara init --gitlab-ci
    ```

    This creates `.gitlab-ci.yml`, `.scripts/kubara-catalog-update.sh`, and `.scripts/install-oras.sh`.
    Pin the installer download URL to a reviewed kubara revision.
    If `.gitlab-ci.yml` already exists, generate the pipeline in a separate working directory and merge the job into your existing pipeline.

    Use a Linux Docker runner. Create a masked `KUBARA_UPDATE_TOKEN` project access token in **Settings → CI/CD → Variables**.
    Give it `api` and `write_repository` scopes and permission to push the automation branch.
    If the variable is protected, the default branch must also be protected.

    Select **Build → Pipelines → New pipeline** on the default branch.
    The job uses the standard `CI_*` variables for the project, API URL, and target branch.

Both examples use `automation/kubara-catalog-update` and serialize runs that update that branch.
Reserve it for the pipeline. Pin the CLI version and container image according to your project's dependency policy.

## Step 3: review generated changes

The pipeline:

1. Selects the highest stable catalog versions without downgrading newer pins.
2. Updates the configured catalog references and pulls the artifacts into the local cache.
3. Runs `kubara generate`.
4. Creates or updates a PR or MR when configuration or generated files differ from the default branch.

The proposed changes include `config.yaml`, `platform-configs/`, and `platform-components/`.
Review these files before merging, including a version-only change when rendered manifests remain the same.
Keep secret files outside these paths and ensure rendered output is suitable for Git.

Runs with no changes leave existing PRs and MRs untouched.
Failed catalog downloads or generation stop publication. A generation failure restores `config.yaml`, but may leave partial output in the CI checkout.
The updater uses yq to preserve comments; YAML formatting may change.

## Use a different configuration location

The update script accepts a working directory followed by a configuration filename:

```bash
bash .scripts/kubara-catalog-update.sh environments/production config.yaml
```

Adjust the pipeline command, GitHub change-detection paths and `add-paths`, or the GitLab staging path list to match.

## Use the GitHub action directly

The root `action.yml` installs a release with checksum verification and runs CLI arguments.
It supports Linux and macOS on amd64 and arm64, with Bash, curl, and jq available on the runner.

```yaml
steps:
  - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
  - uses: kubara-io/kubara@main
    with:
      version: latest
      working-directory: .
      args: '["generate", "--helm"]'
```

| Input | Default | Description |
| --- | --- | --- |
| `version` | `latest` | CLI release tag or version |
| `args` | `["generate"]` | JSON array of arguments; `[]` installs only |
| `working-directory` | `.` | Directory in which to run kubara |

Arguments are passed literally, without shell expansion. Subsequent steps can invoke `kubara` directly.
The action reference selects the action implementation; `version` selects the CLI binary independently.
Use a reviewed action commit or release containing this feature.

## Validate a catalog repository

Add optional validation pipelines when creating a catalog:

```bash
kubara catalog create --github-actions my-catalog
kubara catalog create --gitlab-ci another-catalog
kubara catalog create --github-actions --gitlab-ci shared-catalog
```

The flags create `.github/workflows/catalog.yaml` and/or `.gitlab-ci.yml` inside the catalog root.
Use that root as the repository root. Existing directories are never overwritten.

The pipelines load service schemas with `kubara schema --catalog .` and check packaging with `kubara catalog package`.
They do not publish releases or validate every Helm and Terraform configuration.
See [How to create a Catalog](create_catalog.md) for the authoring workflow.
