| status       | date       | decision-makers | consulted   | informed    |
| ------------ | ---------- | --------------- | ----------- | ----------- |
| **proposed** | 2026-09-12 | kubara-Team     | kubara-Team | kubara-Team |

# Disassembly Helm Template Library into Local Helm Manifests

## Context and Problem Statement

The main catalog and external catalogs can use different template library versions.
When the main catalog is upgraded, external catalogs are not upgraded automatically.
Their library references can then point to the wrong version and break the upgrade.

## Decision Drivers

- Catalogs should be upgraded independently
- Catalog upgrades should not break because of shared library versions

## Considered Options

- Keep the shared template library
- Store the required manifests in each Helm chart
- Don't use template library in the external catalogs

## Decision Outcome

Proposed option: **store the required manifests in each Helm chart**.

### Consequences

- **Good**, because catalogs no longer depend on the same template library version
- **Good**, because catalog upgrades are more reliable
- **Bad**, because some manifests are duplicated

### Confirmation

Catalog Helm charts contain no template library dependency or include.


### More Information

Example implementation of storing required manifests in each Helm chart.

- example: https://github.com/la-cc/catalogs/tree/improvement/remove-template-library