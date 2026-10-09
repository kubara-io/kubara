<!--- remove this comment block
status: "{proposed | rejected | accepted | deprecated | … | superseded by ADR-0123}"
date: {YYYY-MM-DD when the decision was last updated}
decision-makers: {list everyone involved in the decision}
consulted: {list everyone whose opinions are sought (typically subject-matter experts); and with whom there is a two-way communication}
informed: {list everyone who is kept up-to-date on progress; and with whom there is a one-way communication}
-->

| status       | date         | decision-makers          | consulted   | informed   |
|:-------------|:-------------|:-------------------------|:------------|:-----------|
| **{status}** | {YYYY-MM-DD} | {list everyone involved} | {experts}   | {informed} |


# Cluster Object Structure ...

## Context and Problem Statement

{Describe the context and problem statement, e.g., in free form using two to three sentences or in the form of an illustrative story. You may want to articulate the problem in form of a question and add links to collaboration boards or issue management systems.}

- SSO fields should be refactored into a more generalized auth/authentication object
- Stage field is currently only a "hint/label" type of field. Its used for pretemplating the DNS name but beyond that doesn't have any current functionality. Might be interesting for future promotion strategies but not as of yet
- DNS field is currently duplicated between top level cluster and terraform block.
- Terraform should be generalized into a generic "infra"/"iac" block to support other methods of setting up environments like cluster-api+crossplane or pulumi or whatever
- The Gateway API PR (#630) already refactors ingressClassName into a new generic networking object:
```
networking:
    ingress:
        className: xxxx
    gateway:
        name: xxxx
        namespace: xxxx
        sectionName: xxxx
```
- dns field should probably be migrated into the new networking block
- catalog and services should stay as top level objects as they are
- argocd will be removed with the new gitops agnostic approach and will be replaced by the config root object gitOps(.engine) and an optional gitOpsOverride



## Decision Drivers

* {decision driver 1, e.g., a force, facing concern, …}
* {decision driver 2, e.g., a force, facing concern, …}

## Considered Options

* {title of option 1}
* {title of option 2}
* {title of option 3}

## Decision Outcome

Chosen option: "**{title of option 1}**", because {justification. e.g., only option, which meets k.o. criterion decision driver | which resolves force {force} | … | comes out best (see below)}.

### Consequences

* **Good**, because {positive consequence, e.g., improvement of one or more desired qualities, …}
* **Bad**, because {negative consequence, e.g., compromising one or more desired qualities, …}

### Confirmation

{Describe how the implementation / compliance of the ADR can/will be confirmed. Is there any automated or manual fitness function? If so, list it and explain how it is applied. Is the chosen design and its implementation in line with the decision? E.g., a design/code review or a test with a library such as ArchUnit can help validate this. Note that although we classify this element as optional, it is included in many ADRs.}

---

## Pros and Cons of the Options

### {title of option 1}

{example | description | pointer to more information | …}

* **Good**, because {argument a}
* **Good**, because {argument b}
* **Neutral**, because {argument c}
* **Bad**, because {argument d}

### {title of other option}

{example | description | pointer to more information | …}

* **Good**, because {argument a}
* **Good**, because {argument b}
* **Neutral**, because {argument c}
* **Bad**, because {argument d}
* …

## More Information

{You might want to provide additional evidence/confidence for the decision outcome here and/or document the team agreement on the decision and/or define when/how this decision should be realized and if/when it should be re-visited. Links to other decisions and resources might appear here as well.}
