# AI Agent Context

Shared guidance for AI code assistants (Claude Code, Codex, Gemini CLI, etc.) working in this repository.

## Scope
These instructions apply to the entire repository unless a deeper `AGENTS.md` overrides them.

## Product Intent (Stable)
- `kubara` is an opinionated, GitOps-first platform CLI.
- The CLI is a single Go binary.
- Core workflow is: init configuration, generate reproducible artifacts, bootstrap Argo CD and platform components.

## Truth Hierarchy
- Stable intent and guardrails: this `AGENTS.md`.
- Actual runtime behavior: implementation + tests in `go-binary/`.
- Usage and process guidance: docs in `docs/content/` and `CONTRIBUTING.md`.

If docs and code diverge, treat code/tests as current behavior and update the nearest relevant docs in the same change.

## Reference Map
- Product setup and bootstrap flow: `docs/content/1_getting_started/bootstrap_process.md`
- Runtime prerequisites: `docs/content/1_getting_started/prerequisites.md`
- Architecture context: `docs/content/4_architecture/architecture_overview.md`
- Contributor and PR workflow: `CONTRIBUTING.md`
- Config schema and template keys: `go-binary/assets/config/configTypes.go` (use `kubara schema` when needed)

## Project Layout
- `go-binary/`: Go CLI implementation, tests, embedded templates, release Makefile
- `docs/`: MkDocs site managed with `uv`
- Root `Makefile`: monorepo entry point delegating to `go-binary/` and `docs/`

## Canonical CLI Flow
1. `kubara init --prep`
2. `kubara init` (or `--overwrite` when updating from `.env`)
3. Optional: `kubara schema`
4. `kubara generate --terraform` and/or `kubara generate --helm`
5. `kubara bootstrap <cluster-name>`

Keep this flow high-level here; detailed provider-specific steps belong in docs.

## Working Style
- Keep changes focused; avoid mixing unrelated fixes.
- Follow existing naming and file organization.
- Prefer small, surgical edits over broad refactors.
- Do not reformat unrelated files.
- Keep code, comments, commit messages, PR titles/descriptions, and issues in English.
- When behavior changes, update the nearest relevant docs in the same change.

## Validation (Smallest Relevant First)
- Go tests: `make test` or `cd go-binary && make test`
- Go build: `make build-binary` or `cd go-binary && make build`
- Docs build: `make docs-build` or `cd docs && make build`
- Docs validation: `make docs-validate`
- Dependency setup: `make install-deps`

## Go Code
- Main module is `go-binary/`.
- Keep compatibility with the Go toolchain declared in `go-binary/go.mod`.
- Prefer table-driven tests when extending test coverage.
- Reuse existing structure under `go-binary/assets/`, `go-binary/cmd/`, `go-binary/templates/`, and `go-binary/utils/`.

## Documentation
- Docs live under `docs/content/`.
- Keep changes aligned with `docs/mkdocs.yml` navigation.
- Preserve relative paths under `docs/content/images/` and `docs/content/assets/`.
- Significant technical or architectural decisions may require an ADR in `docs/content/7_decisions/`.

## Avoid
- Do not introduce new tooling/dependencies without clear need.
- Do not edit generated artifacts unless a source change requires it.
- Do not assume docs and Go changes are independent; check whether both need updates.
