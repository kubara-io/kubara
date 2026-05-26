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
- Actual runtime behavior: implementation + tests in `src/`.
- Usage and process guidance: docs in `docs/content/` and `CONTRIBUTING.md`.

If docs and code diverge, treat code/tests as current behavior and update the nearest relevant docs in the same change.

## Reference Map
- Product setup and bootstrap flow: `docs/content/1_getting_started/bootstrap_process.md`
- Runtime prerequisites: `docs/content/1_getting_started/prerequisites.md`
- Architecture context: `docs/content/4_architecture/architecture_overview.md`
- Contributor and PR workflow: `CONTRIBUTING.md`
- Config schema and template keys: `src/internal/config/types.go` (use `kubara schema` when needed)

## Project Layout
- `src/`: Go CLI implementation, tests, embedded templates, release Makefile
- `docs/`: MkDocs site managed with `uv`
- Root `Makefile`: monorepo entry point delegating to `src/` and `docs/`

## Working Style
- Keep changes focused; avoid mixing unrelated fixes.
- Follow existing naming and file organization.
- Prefer small, surgical edits over broad refactors.
- Do not reformat unrelated files.
- Keep code, comments, commit messages, PR titles/descriptions, and issues in English.
- When behavior changes, update the nearest relevant docs in the same change.
- Use the PR and issue templates; fill required sections and do not remove template structure.

## Validation (Smallest Relevant First)
- Go tests: `make test` or `cd src && make test`
- Go build: `make build-binary` or `cd src && make build`
- Docs build: `make docs-build` or `cd docs && make build`
- Docs validation: `make docs-validate`
- Dependency setup: `make install-deps`

## Go Code
- Main module is `src/`.
- Keep compatibility with the Go toolchain declared in `src/go.mod`.
- Prefer table-driven tests when extending test coverage.
- Reuse existing structure under `src/cmd/` and `src/internal/`.

## Documentation
- Docs live under `docs/content/`.
- Keep changes aligned with `docs/mkdocs.yml` navigation.
- Preserve relative paths under `docs/content/images/` and `docs/content/assets/`.
- Significant technical or architectural decisions may require an ADR in `docs/content/7_decisions/`.

## Avoid
- Do not introduce new tooling/dependencies without clear need.
- Do not edit generated artifacts unless a source change requires it.
- Do not assume docs and Go changes are independent; check whether both need updates.
