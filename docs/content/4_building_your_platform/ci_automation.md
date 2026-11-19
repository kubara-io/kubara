# CI Automation

This guide explains how to run `kubara generate` automatically in GitHub Actions and GitLab CI.

The pipeline keeps your GitOps repository in sync:

1. When a pull request or merge request changes `config.yaml`, the pipeline runs `kubara generate` and commits the updated manifests (including schema migrations written back to `config.yaml`) back to the branch.
2. A daily scheduled run checks the default branch for drift and opens a pull request or merge request if changes are detected.

## Adding the Pipeline

Use `kubara init` with the flag for your platform. Existing CI files are always preserved.

=== "GitHub Actions"

    ```bash
    kubara init --github-actions
    ```

    This creates `.github/workflows/kubara-generate.yml`.

    In repository settings under **Actions > General**, enable **Allow GitHub Actions to create and approve pull requests**.
    The workflow requires `contents: write` and `pull-requests: write` permissions.

    By default, changes are committed using `GITHUB_TOKEN`. Because GitHub does not trigger downstream workflow runs from `GITHUB_TOKEN` pushes, repositories that require validation or status checks on generated commits can authenticate via a GitHub App. Install the App on the repository with `contents: write` and `pull-requests: write` permissions, set the `APP_CLIENT_ID` repository variable and `APP_PRIVATE_KEY` secret, and uncomment the `Create automation token` step in `.github/workflows/kubara-generate.yml`.

=== "GitLab CI"

    ```bash
    kubara init --gitlab-ci
    ```

    This creates `.gitlab/kubara.gitlab-ci.yml` and includes it in `.gitlab-ci.yml` (or preserves an existing `.gitlab-ci.yml` and logs instructions to include it).

    For automatic merge request creation on scheduled drift runs, ensure your CI token or project access token has repository write permissions.
    By default, the pipeline uses `$CI_JOB_TOKEN`. Pushing with `$CI_JOB_TOKEN` requires enabling **Allow Git push requests to the repository** in your project's CI/CD Job token permissions (note that pushes made with `$CI_JOB_TOKEN` do not trigger a new pipeline). If your `$CI_JOB_TOKEN` does not have write access to the repository, configure a project or group access token with repository write permissions and provide it via the masked CI/CD variable `KUBARA_UPDATE_TOKEN`.

## Updating Catalog Versions

Catalog versions in `config.yaml` can be updated in two ways:

1. **Renovate (Recommended)**: `kubara init` generates a `renovate.json` file. When Renovate runs, it discovers new catalog releases and opens a pull request. The CI workflow detects the version bump, runs `kubara generate`, and commits the rendered platform manifests to the pull request.
2. **Manual bumps**: Change the catalog version tag in `config.yaml` on a branch and push. The pipeline generates the manifests and commits them to your branch for review.

## Private OCI Catalogs

For security, `kubara generate` executes with an isolated environment and clean `$HOME` directory to prevent secrets from leaking into generated platform manifests.

If your configuration references catalogs from a private OCI registry, authenticate before running generation. The pipeline automatically forwards `$HOME/.kubara/credentials.json` into the isolated environment:

=== "GitHub Actions"

    Add a step before generation that runs [`kubara catalog login`](../2_concepts/catalog_distribution.md#step-3-log-into-the-registry) (replace `ghcr.io` with your private registry):

    ```yaml
    - name: Authenticate with catalog registry
      run: |
        echo "${{ secrets.CATALOG_REGISTRY_TOKEN }}" | kubara catalog login \
          --username "${{ secrets.CATALOG_REGISTRY_USER }}" \
          --password-stdin \
          ghcr.io
    ```

=== "GitLab CI"

    Add [`kubara catalog login`](../2_concepts/catalog_distribution.md#step-3-log-into-the-registry) to `before_script` in your `.gitlab-ci.yml`:

    ```yaml
    kubara-generate:
      before_script:
        - apk add --no-cache git
        - >-
          echo "$CI_REGISTRY_PASSWORD" | /kubara catalog login \
            --username "$CI_REGISTRY_USER" \
            --password-stdin \
            "$CI_REGISTRY"
    ```

## Using the GitHub Action Directly

The scaffolded generation workflow relies on GitHub-specific features (such as `gh` CLI for PR comments), but the standalone `kubara-action` has no GitHub-specific dependencies and is also compatible with Forgejo Actions:

```yaml
steps:
  - uses: actions/checkout@v4
  - uses: kubara-io/kubara-action@7525333662824339b4df949ef5e5e82a518cae65 # v0.1.0
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
