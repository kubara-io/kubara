# GitOps, Platform Engineering, and IDPs

This page gives a short introduction to the ideas behind kubara:

- GitOps
- Platform Engineering
- Internal Developer Platforms (IDPs)

You do not need to know every term before using kubara, but these ideas make it easier to understand why kubara is built the way it is.

## What is GitOps?

GitOps is an operating model where Git is the main source of truth for your platform and application configuration.

In practice, that means:

- You describe the desired state in files
- You store those files in Git
- Changes happen through commits and pull requests
- A controller such as Argo CD applies and reconciles that desired state

The goal is to make changes:

- Reviewable
- Repeatable
- Auditable
- Easier to roll back

kubara follows this model closely. It generates artifacts that are meant to be committed to Git and then reconciled by Argo CD.

Useful external resources:

- [OpenGitOps](https://opengitops.dev/)
- [What is GitOps? - Red Hat](https://www.redhat.com/en/topics/devops/what-is-gitops)
- [Argo CD documentation](https://argo-cd.readthedocs.io/)

## What is Platform Engineering?

Platform Engineering is the practice of building and operating a reusable platform that helps application teams move faster.

Instead of every team solving the same infrastructure and delivery problems on its own, a platform team creates shared building blocks such as:

- Cluster setup
- GitOps workflows
- Networking defaults
- Observability
- Security controls
- Self-service deployment paths

The goal is not only standardization. The goal is also to make the secure and maintainable path the easiest path.

kubara fits here as a CLI that helps platform engineers bootstrap and manage that shared foundation.

Useful external resources:

- [PlatformEngineering.org](https://platformengineering.org/)
- [CNCF Platform White Paper](https://tag-app-delivery.cncf.io/whitepapers/platforms/)
- [CNCF Platform Engineering Maturity Model](https://tag-app-delivery.cncf.io/whitepapers/platform-eng-maturity-model/)

## What is an Internal Developer Platform (IDP)?

An Internal Developer Platform, or **IDP**, is the product that platform teams build for internal users such as developers, operators, and service teams.

An IDP usually gives teams a simpler way to do things like:

- Deploy workloads
- Request infrastructure
- Use shared services
- Follow security and policy standards
- Onboard new environments

An IDP is not only a Kubernetes cluster. It is the full internal platform experience around that cluster.

In many organizations, the IDP includes:

- Workflows
- Templates
- Automation
- Guardrails
- Documentation
- Self-service interfaces

kubara helps you build the lower-level and mid-level parts of that platform foundation: cluster bootstrapping, reusable platform setup, GitOps structure, and catalog-based packaging of platform architecture.

## How these ideas connect

These three ideas work well together:

1. **Platform Engineering** builds and owns the shared platform.
2. They use **GitOps** as their main way of delivering platform changes safely.
3. And the **IDP** is the internal product that the Platform Engineers produce to be used by the application teams.

kubara sits mainly on the platform side:

- It helps platform engineers define and generate the platform
- It packages reusable platform architecture as catalogs
- It utilizes GitOps tools such as Argo CD for the actual delivery and reconciliation

## Where to go next

- For the overall kubara model: [Overview](overview_core_concept.md)
- For the catalog model: [Catalogs](catalogs.md)
- For catalog distribution: [Catalog distribution](catalog_distribution.md)
- For workload onboarding on top of the platform: [Workload Onboarding with Argo CD](../5_workload_onboarding/overview.md)
