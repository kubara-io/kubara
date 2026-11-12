# CI Automation

This guide explains how to run `kubara generate` automatically in GitHub Actions and GitLab CI.

The pipeline keeps your GitOps repository in sync:

1. When a pull request or merge request changes `config.yaml`, the pipeline runs `kubara generate` and commits the updated manifests back to the branch.
2. A daily scheduled run checks the default branch for drift and opens a pull request or merge request if changes are detected.

## Adding the Pipeline

Use `kubara init` with the flag for your platform. Existing files are preserved unless `--overwrite` is set.

=== "GitHub Actions"

    ```bash
    kubara init --github-actions
    ```

    This creates `.github/workflows/kubara-generate.yml`.

    In repository settings under **Actions > General**, enable **Allow GitHub Actions to create and approve pull requests**.
    The workflow requires `contents: write` and `pull-requests: write` permissions.

=== "GitLab CI"

    ```bash
    kubara init --gitlab-ci
    ```

    This creates `.gitlab-ci.yml`.

    For automatic merge request creation on scheduled drift runs, ensure your CI token or project access token has repository write permissions.

## Updating Catalog Versions

Catalog versions in `config.yaml` can be updated in two ways:

1. **Renovate (Recommended)**: `kubara init` generates a `renovate.json` file. When Renovate runs, it discovers new catalog releases and opens a pull request. The CI workflow detects the version bump, runs `kubara generate`, and commits the rendered platform manifests to the pull request.
2. **Manual bumps**: Change the catalog version tag in `config.yaml` on a branch and push. The pipeline generates the manifests and commits them to your branch for review.

## Using the GitHub Action Directly

You can use the root `kubara` action in your own GitHub workflows:

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: kubara-io/kubara@main
    with:
      version: latest
      args: '["generate"]'
      working-directory: .
```

| Input | Default | Description |
| --- | --- | --- |
| `version` | `latest` | CLI release tag or version. |
| `args` | `["generate"]` | JSON array of CLI arguments. Use `[]` to install only. |
| `working-directory` | `.` | Working directory for the command. |

## Validating Catalog Repositories

When creating a new catalog, you can include validation workflows:

```bash
kubara catalog create --github-actions my-catalog
kubara catalog create --gitlab-ci my-catalog
```

The generated pipeline validates catalog schemas with `kubara schema --catalog .` and tests packaging with `kubara catalog package`.
