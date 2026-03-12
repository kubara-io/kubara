# Core Concept

`Kubara` is a framework and bootstraping tool for building and operating a production-grade Kubernetes platform. 
It is built to be a modular and opinionated foundation.  

It provides a comprehensive, ready-to-extend framework that bundles
everything needed to design, deploy and scale internal platform capabilities
- from infrastructure automation to developer self-service.
This framework is not a one-size-fits-all solution. Instead, it establishes
standardized `building blocks`, `patterns` and `principles` to accelerate and
unify platform development across teams and environments.


<p align="center">
  <img src="../../images/architecture-overview.jpg" alt="Kubara Architecture Overview" width="700"/>
</p>

## Directory Structure

Kubara generates a specific directory structure in your Git repository to separate concerns:

- **`managed-service-catalog/`**
This directory contains the reusable components (Terraform modules and Helm charts) provided and maintained by Kubara. 
You should generally not modify files in this directory, as they may be updated with new Kubara releases.


- **`customer-service-catalog/`**
It contains cluster-specific configurations and your custom values for the kubara setup.


## Architecture
The Diagramm shows a typical Kubara Workflow when following the [manual](bootstrap_process.md)

``` mermaid
graph TD
    A[⚙️ Set .env-File & config.yaml] --> B;
    B[🤖 'Kubara generate' <br> generates tf & charts] --> C;
    C[📤 Commit & Push to Git] --> D{Apply Terraform?};
    D -- Yes --> G[☁️ Apply Cloud Resources];
    G --> I[🔑 Apply kubeconfig & 'kubara bootstrap <cluster-name>'];
    D -- No --> I;
    I --> F[Enjoy your Kubara Deployment 🎉];
```

1. Platform Engineer must set parameters in config files (⚠️Caution: Environment variables have priority over config values)
2. "Kubara generate" templates and creates Terraform & Umbrella Helm-Charts.   
Now you should commit and push your templates to your git and optionally apply Terraform
3. `kubara bootstrap <cluster-name>` rolls out Argo CD and required CRDs to your control plane cluster.
4. Secrets are synced via External Secrets based on your configured SecretStore/ClusterSecretStore.
Argo CD manages itself and rolls out all [generated Helm Charts](../3_components/components_overview.md).




## What's included

- **Helm Charts**
  Predefined, customizable modules for core components like ingress,
  observability, identity, policies, CI/CD, and app lifecycles.

- **Architecture Models**
  Reference topologies for single-cluster, multi-cluster (hub & spoke)
  and hybrid cloud setups.

- **Templates & Reusable Patterns**
  YAML templates, GitOps folder structures, RBAC models and security
  best practices.

- **Operational Playbooks**
  Step-by-step guides for provisioning, upgrades, disaster recovery,
  secrets management and policy enforcement.

- **Documentation & Decision Records**
  Well-documented reasoning behind design choices (ADR format), technical
  constraints, and usage guidelines.

- **Extension Guidelines**
  Conventions and interfaces to integrate custom workloads, controllers or
  cluster add-ons in a maintainable way.

## Core Goals

- Enable rapid platform rollout
- Standardize architecture and governance
- Ensure security, compliance and observability
- Empower teams through self-service and GitOps

## Adding new tools 
If the current toolset doesn't meet your needs or is missing key features, you can propose new tools [here](https://github.com/kubara-io/kubara/blob/main/CONTRIBUTING.md#integration-requirements-catalogue).
