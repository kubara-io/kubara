# Architecture Decision Records (ADR)

## What is an ADR?

An **Architecture Decision Record (ADR)** documents a **significant technical/architectural decision** in a lightweight and structured way — including context, the decision itself, and its consequences.

An Architectural Decision (AD) is a justified software design choice that addresses a functional or non-functional requirement of architectural significance.  
(see:  [Markdown Architectural Decision Records](https://adr.github.io/madr/) ) 

**Goal:** Improve **traceability**, avoid repeating discussions, and make decisions understandable long after they were made.

---

## When do we write an ADR?

Write an ADR when a decision:

- has **long-term impact** on the platform or architecture
- affects **core components / shared standards / interfaces**
- has **multiple valid alternatives** that need comparison
- impacts **security, scalability, reliability, maintainability**
- introduces or replaces a **major technology/tooling choice**
- intentionally **deviates from an existing guideline/pattern**

ADRs are **not required** for small refactorings, minor implementation details, or purely operational changes with no architectural impact.

---

## Where do we store ADRs?

ADRs are stored as **Markdown files in the repository** (versioned like code).

**Location:** `docs/content/7_decisions/`

This ensures ADRs are:
- reviewable via PR
- searchable in the codebase
- permanently available alongside the implementation

---

## ADR numbering & naming

Each ADR receives a **unique sequential number** to ensure stable references.

### Rules
- Numbering starts at `ADR-0001`
- Numbers are **never reused**
- The number is assigned **when the ADR is created** (proposal stage)
- Even **superseded / deprecated** ADRs keep their original number

### File naming convention
```
ADR-<number>-<short-decision-title>.md
```

Example:
```
ADR-0001-replace-deprecated-nginx-ingress.md
```

**Title guidelines:**
- lowercase
- hyphen-separated
- short and descriptive

---

## ADR template

To create a new ADR:  

1. Copy `ADR-template.md` [ADR-template](ADR-template.md)
2. Rename it using the naming convention
3. Fill in all sections
4. Set the initial status to `proposed`

---

## How we use ADRs in kubara

### Process (validated)

**Create proposal**
   - Engineer creates an ADR using the template
   - Status: `proposed`

**Discuss and decide**
   - ADR is reviewed and discussed in the **maintainer ADR meeting**
   - Decision is documented inside the ADR  
   - Status becomes: `accepted` (if approved)

**Implement**
   - Implementation starts **after acceptance**
   - Changes are delivered via Pull Requests  
   - Optional status during implementation: `active`

**Merge and persist**
   - ADR is merged into the repository (usually via PR)
   - ADR becomes part of the official project documentation


✅ This ensures the ADR captures the decision **before or during implementation**, not only afterwards.

---

## ADRs and Pull Requests / Issues

- ADRs **can be created and reviewed via PR** (recommended)
- An Issue (bug/feature request) is **optional**, but useful for linking background context
- Best practice: link related PRs/issues in the ADR for traceability

---

## ADR overview (ADR Index)

To keep ADRs discoverable, we maintain an **ADR index page** with a quick overview of 
ADR number with link to the document, title and status. 

**Index page location:** [ADR-index](ADR-index.md) -> `docs/content/7_decisions/ADR-index.md` 

The index can be maintained:
- **manually** (simple table, best for a small number of ADRs), or
- **automatically** via a small script in CI (recommended once the list grows)

---

## Status lifecycle

We use the following ADR status values:

- `proposed` – under discussion
- `accepted` – approved decision
- `active` – currently being implemented (optional)
- `superseded` – replaced by a newer ADR
- `deprecated` – no longer valid but kept for traceability

---

## Quick checklist (author)

Before opening the PR, ensure:
- [ ] file name follows `ADR-000X-...`
- [ ] status is set to `proposed`
- [ ] decision context is clear
- [ ] alternatives are documented
- [ ] consequences are explicitly stated
- [ ] references/links are included (optional but recommended)

## See also
For creating proposals for new tools see [here](https://github.com/kubara-io/kubara/blob/main/CONTRIBUTING.md#integration-requirements-catalogue).
