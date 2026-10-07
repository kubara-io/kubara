| status       | date       | decision-makers | consulted          | informed           |
|--------------|------------|-----------------|--------------------|--------------------|
|              |            |                 |                    |                    |

# Directory-Scoped Execution Context for Multi-Hub GitOps Repositories

## Context and Problem Statement

Following feedback and bugreports from our community around the handling and documentation of multi-hub setups, 
we decided to improve kubaras support for multi-hub setups. 

Previously, `kubara generate` hardcoded its output paths to `./platform-components` and `./platform-configs/<cluster>` 
relative to the current working directory, and wiped `./platform-components` before every generate run. 
When multiple configuration files coexisted in a repository:
1. Running generation for one config would wipe or overwrite the shared `platform-components` of another config, possibly resulting in the deletion of required files.
2. If different configs referenced different catalog versions or customized services, conflicts occurred.
3. Argo CD Application manifests need accurate repository-relative subpaths to locate charts and values overlays within the Git repository.
4. Relative paths in Terraform templates (e.g. `../../../../platform-components/...`) depend on a consistent relative depth between `platform-configs` and `platform-components`.

Kubara needs an ergonomic, deterministic way to support multiple hub-and-spoke configurations within the same repository without cross-configuration interference.

## Decision Drivers

- Different configurations may use different catalog versions and platform stacks.
- Templating for one configuration must never mutate, wipe or overwrite another configuration's manifests.
- Zero breaking changes to existing single-configuration repositories.
- Keep Terraform relative module paths (`../../../../platform-components/...`) working without requiring modifications to catalog templates.
- Git repository subpaths must be computed automatically to simplify Argo CD GitOps workflows.
- Developer experience must be intuitive: Easily target individual hubs, a set of hubs or run batch generation across all.
- Minimize sources of mixups in execution context around .env file and config.yaml

## Decision Outcome

**One Hub per Config, one Folder per Hub**.

### Folder Layout
Hubs exist in isolated directories, at the root level or nested (e.g. `prod`, `project-a/staging`) containing:
- `config.yaml`
- `.env`
- `platform-components/` (isolated generated Helm charts and Terraform modules)
- `platform-configs/` (cluster-specific overlays)

Because `platform-configs` and `platform-components` remain siblings inside each hub directory, existing Terraform relative source paths (`../../../../platform-components/...`) remain valid.

### Automatic GitOps Path Computation
When executing within a hub folder, `kubara generate` automatically:
1. Traverses up the directory tree to discover the Git repository root (`.git`).
1. Calculates the Git-relative subpath from the Git root to the hub (e.g. `setups/dev-fleet`).
1. Injects computed repository paths into `argocd.repo`:
   - `components.path`: `<git-rel-path>/platform-components/helm`
   - `configs.path`: `<git-rel-path>/platform-configs`

### CLI Ergonomics
1. **Targeting**: `generate --hub <path>` sets the target hub directory, accepting a list of hubs to generate.
1. **Directory Scoping**: Commands execute within the hub directory or target hubs via `--hub`. Legacy flags (`--work-dir`, `--config-file`, and `--env-file`) are removed in favor of directory-scoped context.
1. **Batch Generation**: `kubara generate --all` (alias `-A`) discovers all configurations in the current directory and generates each hub in its own isolated context.
1. **Removal of old Flags**: `--work-dir`, `--config-file` and `--env-file` have been removed and are replaced by `--hub`

## Consequences

- **Good**, because multiple hub-and-spoke setups can live in the same Git repository without file collisions.
- **Good**, because setups can independently upgrade catalog versions and dependencies.
- **Good**, because Argo CD path calculation requires zero manual configuration.
- **Good**, because single-config repositories at root remain completely unaffected (zero breaking changes).
