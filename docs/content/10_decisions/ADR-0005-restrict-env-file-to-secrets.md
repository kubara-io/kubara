| status       | date       | decision-makers | consulted          | informed           |
|--------------|------------|-----------------|--------------------|--------------------|
| **proposed** | 2026-09-23 | kubara-Team     |                    |                    |

# Restrict env file to secrets

## Context and Problem Statement

kubara currently uses both `.env` and `config.yaml` to define platform parameters. This split creates confusion over which file owns which setting:

- Flat cluster configuration (`PROJECT_NAME`, `PROJECT_STAGE`, `ARGOCD_GIT_URL`, `ARGOCD_GIT_AUTH_MODE`, `ARGOCD_HELM_REPO_URL`) lives in `.env`. What does this apply to? Just the Hub? Spokes as well?
- Initializing a project requires a two-step command flow: Running `kubara init --prep` to generate `.env`, editing it, then running `kubara init` to parts of the values into `config.yaml`.
- Later runs using `kubara init --overwrite` read `.env` to overwrite values in `config.yaml` (in-memory), maintaining two conflicting sources of truth as the config might state something different than the `.env` file.
- `kubara generate` loads `.env` and injects it into template contexts under `.env`, creating the risk of leaking secrets into rendered GitOps manifests committed to version control. This MUST be removed.

We need a clear boundary: `config.yaml` SHOULD be the sole source of truth for platform configuration, and `.env` SHOULD only contain credentials required for cluster bootstrapping.

## Decision Drivers

- Single source of truth for all cluster and platform configuration.
- Zero risk of secret leakage into generated GitOps manifests.
- Simpler initialization flow without two commands.
- Clearer separation between declarative git-tracked configuration and local credentials.

## Considered Options

- Keep the existing two-staged command workflow and active refresh of non-secret fields from `.env` to `config.yaml`.
- Move everything into `config.yaml` using external secret references or inline encryption using `age` or `sops`.
- Restrict `.env` strictly to secrets/credentials solely for bootstrapping. Only use `config.yaml` for generation and fully decoupled from `.env`.

## Decision Outcome

Chosen option: ** restrict `.env` strictly to secrets/credentials solely for bootstrapping. Only use `config.yaml` for generation and fully decoupled from `.env`.**

### Configuration boundary

`config.yaml` is the ONLY source of truth for cluster metadata, repository access and service parameters:

- `clusters[].name` replaces `PROJECT_NAME`.
- `clusters[].stage` replaces `PROJECT_STAGE`.
- `clusters[].argocd.repo.authMode` replaces `ARGOCD_GIT_AUTH_MODE`.
- `clusters[].argocd.repo.git.*.url` replaces `ARGOCD_GIT_URL` and `ARGOCD_GIT_HTTPS_URL`.
- `clusters[].argocd.helmRepo.url` replaces `ARGOCD_HELM_REPO_URL`.

`.env` will only contain the core secrets and auth credentials necessary during bootstrap:

- Passwords and tokens: `ARGOCD_WIZARD_ACCOUNT_PASSWORD`, `ARGOCD_GIT_PAT_OR_PASSWORD`, `ARGOCD_HELM_REPO_PASSWORD`.
- Private keys: `ARGOCD_GIT_SSH_PRIVATE_KEY`, `ARGOCD_GIT_GITHUB_APP_PRIVATE_KEY`.
- App credentials and enterprise URLs lacking a config equivalent: `ARGOCD_GIT_GITHUB_APP_ID`, `ARGOCD_GIT_GITHUB_APP_INSTALLATION_ID`, `ARGOCD_GIT_GITHUB_APP_ENTERPRISE_BASE_URL`.
- Auth usernames tied to secret credentials: `ARGOCD_GIT_USERNAME`, `ARGOCD_HELM_REPO_USERNAME`.
- Registry pull credentials: `DOCKERCONFIG_BASE64`.

If an existing `.env` contains deprecated non-secret fields, kubara logs a warning and ignores them.

### Generation boundary

`kubara generate` stops reading `.env` entirely:

- The `.env` context is removed from the template rendering context to ensure NO secrets can leak into Git manifests.
- Templates must source all cluster and service values from `.cluster` and `.catalog`.
- Templates attempting to reference `.env` will fail at render time. This is a BREAKING CHANGE!

### Streamlined initialization

`kubara init` replaces the multi-step flow:

- The `--prep` flag is removed.
- The `--overwrite` flag is removed.
- Running `kubara init` creates any missing files (`config.yaml` and `.env`) with initial scaffold templates in a single step.
- If both files already exist, `kubara init` fails because there is nothing to do.
- `kubara init --local` generates both files pre-filled with local development settings in one step.

### Multi-cluster secret resolution

Keys in `.env` remain flat. During `kubara bootstrap --cluster <name>`, kubara reads `.env` and applies credentials to the targeted cluster. Distinct cluster credentials across environments use separate dotenv files via `--env-file`.

### Consequences

- **Good**, because `config.yaml` becomes the single source for repository and cluster settings.
- **Good**, because `kubara generate` cannot leak secrets into git-tracked manifests.
- **Good**, because `kubara init` no longer requires running `--prep` first.
- **Good**, because removing `--overwrite` prevents accidental loss of configuration edits.
- **Neutral**, because existing scripts referencing `kubara init --prep` or `kubara init --overwrite` must be updated.
- **Bad**, because any custom templates expecting `.env` in their context will break.

## Non-Goals

- Replacing `.env` with a built-in secret manager or vault client.
- Encrypting secrets within git-tracked files.
- Supporting per-cluster secret namespaces inside a single `.env` file.
