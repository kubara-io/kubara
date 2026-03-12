# Contributing to Kubara

Thank you for your interest in contributing to Kubara.
We welcome bug reports, documentation improvements, and code changes.

## Contributor vs. Maintainer

- Contributors: anyone submitting issues, ideas, docs, or code changes.
- Maintainers: core team members reviewing, approving, and merging changes.
- Maintainer details: [docs/content/5_community/maintainers.md](./docs/content/5_community/maintainers.md)

## Reporting Issues

Open issues here: <https://github.com/kubara-io/kubara/issues>

Please include:

- What happened
- Steps to reproduce
- Expected vs. actual behavior
- Logs/screenshots where relevant

## Before You Start

Before implementing a bug fix or feature, create or discuss an issue first.
For larger topics or architectural changes, align on the approach before implementation.
If a major technical decision is needed, document it with an ADR:
[docs/content/7_decisions/ADR.md](./docs/content/7_decisions/ADR.md).

## Development Setup

Requirements:

- Go `1.25.7` (see `go-binary/go.mod`)
- Git
- `pre-commit` (recommended)

Setup:

```bash
git clone https://github.com/kubara-io/kubara.git
cd kubara
pre-commit install --install-hooks
```

## Build and Validate

```bash
cd go-binary
go test ./...
go build -o kubara .
pre-commit run --all-files
```

## Commit and PR Guidelines

### One PR per topic

Keep pull requests focused on one change topic.
Avoid bundling unrelated changes in one PR.

### PR title convention

Use Conventional Commits style:

- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `refactor:` for internal code improvements
- `chore:` for tooling/maintenance/CI tasks

### PR description requirements

Fill the PR template completely and link related issues.
A good PR should clearly explain:

- What changed
- Why it changed
- How it was tested
- Any rollout or review notes

### Review etiquette

- Be open to feedback and iterate quickly.
- Respond to comments in a timely manner.
- Keep discussion focused on the specific change.

## Branch Strategy

- Create a dedicated feature branch for each change.
- Keep your branch up to date with the target integration branch.
- Use small, reviewable commits.

## Integration Requirements Catalogue

If you propose adding a new tool/component to Kubara, include a structured proposal.

### Strategic Alignment Criteria (40%)

- Core mission fit and community alignment
- Architecture compatibility and API consistency
- Dependency footprint and security posture

### Operational Impact Assessment (30%)

- Maintenance overhead, release cadence, and security support
- Integration complexity, migration path, and rollback strategy

### Value Proposition Criteria (30%)

- Problem/pain-point resolution and measurable value
- Uniqueness vs. existing stack capabilities
- Reasonable total cost of ownership and learning curve

### Evaluation Process

Include:

- Problem statement
- Alternatives analysis
- Implementation plan
- Success metrics

Review stages:

- Initial screening
- Technical evaluation
- Pilot testing (if needed)
- Final maintainer decision

### Decision Matrix

- Auto-approve: score >= 85/100 without critical failures
- Conditional approve: score 70-84/100 with mitigation plan
- Reject: score < 70/100 or critical failure

### Automatic Rejection Criteria

- Duplicate functionality without clear benefit
- Proprietary dependencies or lock-in
- No viable migration/rollback strategy
- Unacceptable maintenance burden

## Documentation Changes

- Update docs when behavior or interfaces change.
- Keep links in `README.md` and docs navigation up to date.

## Code of Conduct

By participating, you agree to follow the project code of conduct:
[docs/content/6_reference/code_of_conduct.md](./docs/content/6_reference/code_of_conduct.md)

## License

By contributing, you agree to the repository licensing model:

- Software/code contributions are licensed under Apache 2.0 ([LICENSE](./LICENSE)).
- Documentation contributions are licensed under CC BY 4.0 ([LICENSE-docs](./LICENSE-docs)).
