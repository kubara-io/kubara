# Migration Guides

This section documents breaking changes and migration steps for upgrading kubara across major or minor versions that require user intervention.

---

## Migrating from v0.18.x to v0.19.0

### Background & Responsibility Cleanup

In kubara v0.18.0 and earlier, configuration responsibilities were split between `.env` and `config.yaml`:

- Running `kubara init --prep` was required to scaffold an initial `.env` file containing cluster parameters (`PROJECT_NAME`, `PROJECT_STAGE`, repository URLs, and credentials).
- Running `kubara init` (or `kubara init --overwrite`) copied those values from `.env` into `config.yaml`, creating conflicting sources of truth.
- `kubara generate` had access to `.env` in its templating context, introducing risk of secret leakage into generated Git manifests.

**Starting with v0.19.0 ([ADR-0005](../10_decisions/ADR-0005-improve-env-var-handling.md)):**

1. **`config.yaml` is the sole source of truth** for all platform and cluster configuration (cluster names, stages, repository URLs, DNS names, and service settings).
2. **`.env` is strictly optional and secret-only**, restricted to bootstrap repository credentials (`GITOPS_*`, `DOCKERCONFIG_BASE64`, and optional `ARGOCD_PASSWORD`).
3. **OS environment variables take precedence**, allowing native CI/CD execution without writing a `.env` file to disk.
4. **`kubara generate` no longer reads `.env`**, ensuring secrets cannot leak into Git manifests.
5. **`kubara init` is a single-step command** that scaffolds `config.yaml`, `env.example`, and `renovate.json` without requiring `--prep` or `--overwrite`.

---

### Migration Steps for Existing Repositories

If you maintain an existing repository initialized with kubara v0.18.0 or earlier:

1. **Verify your `config.yaml`:**
   Ensure each cluster in `spec.clusters` has explicit `name`, `stage`, and `argocd.repo.git` URLs. These are no longer read from `.env`.

2. **Clean up your `.env` (if used):**
   - Remove non-credential variables (`PROJECT_NAME`, `PROJECT_STAGE`, `ARGOCD_GIT_URL`, `ARGOCD_HELM_REPO_URL`).
   - Rename any legacy `ARGOCD_GIT_*` or `ARGOCD_HELM_*` credential variables to their `GITOPS_*` equivalents
   - If you previously configured `ARGOCD_WIZARD_ACCOUNT_PASSWORD`, rename it to `ARGOCD_PASSWORD` or remove it entirely to let kubara autogenerate a secure password during bootstrap.

3. **Update CI/CD workflows:**
   - Remove `kubara init --prep` and `kubara init --overwrite` flags from your pipelines. `kubara init` runs in a single step.
   - Update any pipeline environment variables from `ARGOCD_GIT_*` to `GITOPS_*`.
   - Ensure `config.yaml` modifications in CI target the YAML file directly (e.g. via scripts or `yq`), rather than relying on `.env` overrides.
