| status       | date       | decision-makers | consulted          | informed           |
|--------------|------------|-----------------|--------------------|--------------------|
| **proposed** | 2026-09-01 | kubara-Team     |                    |                    |

# Support multiple GitOps engines, top-level GitOps configuration, and fleet flavors

## Context and problem

Kubara originally tied everything directly to Argo CD and put the settings inside each cluster entry (`clusters[].argocd`):
- The cluster configuration required an `argocd` block (`src/internal/config/types.go:65`).
- Environment variables specifically named Argo CD, such as `ARGOCD_GIT_HTTPS_URL` and `ARGOCD_WIZARD_ACCOUNT_PASSWORD` (`src/internal/envconfig/env.go:39-42`).
- The catalog code hardcoded `BootstrapServiceArgoCD = "argo-cd"` (`src/internal/catalog/bootstrap.go:10`).
- The bootstrap command directly checked for Argo CD pods and deployments (`src/internal/cmd/bootstrap/bootstrap.go:178-186`).
- Git repository secrets used hardcoded Argo CD labels (`argocd.argoproj.io/secret-type: repository`, `src/internal/cmd/bootstrap/secrets.go:117-120`).

### Platform rule
kubara always follows one rule for clusters:
> **A configuration has exactly one `hub` cluster and zero or more `spoke` clusters.**

This brings up a question: **Should GitOps settings sit at the top of the file (`gitops:`) or stay inside each cluster entry (`clusters[].gitops`)?**

We also want to support other tools like [Flux v2](https://fluxcd.io/flux/concepts/), [Projectsveltos](https://projectsveltos.github.io/sveltos/), and [argocd-agent](https://github.com/argoproj-labs/argocd-agent). To do that cleanly, kubara needs a general way to handle GitOps without tying Go code to any single tool.

## Goals

1. **Match the single-hub setup.** The GitOps engine runs on the hub cluster. Global settings like repository URLs and credentials belong at the top level.
2. **Remove repeated settings.** Stop copying the same repository URLs and branch names into every spoke cluster.
3. **Support multiple engines.** Let users choose their engine, like `engine: argocd` or `engine: flux`.
4. **Follow catalog boundaries.** Keep tool-specific logic in catalog files and templates, not in Go code (`ADR-0002`).
5. **Support fleet flavors.** Allow extra tools, such as Sveltos for add-ons or Argo CD Agent for edge clusters, without breaking the configuration format.
6. **Clean bootstrap process.** Install, configure, and check the health of any engine using catalog metadata.

---

## Evaluation: top-level `gitops:` vs per-cluster `clusters[].gitops`

### 1. Does a spoke cluster need GitOps settings?

Looking at how GitOps tools work in multi-cluster setups:

| Setup | Hub role | Spoke role | What the spoke needs in config |
| :--- | :--- | :--- | :--- |
| **Standard push** (Argo CD ApplicationSet, Flux remote kubeconfig) | Runs the engine, keeps repo secrets, and applies manifests directly to spokes using remote kubeconfigs. | Passive target. Runs no GitOps controllers. | **Nothing.** The hub only needs cluster access secrets. |
| **Agent mode** (Argo CD Agent in managed mode) | Runs Argo CD and the agent principal server. Sends sync jobs to spokes. | Runs a small agent that calls home to the hub over mTLS. | **Derived automatically.** Spoke connection info comes from hub DNS and certificates. |
| **Autonomous edge** (Autonomous Argo CD Agent or local Flux) | Runs the main dashboard and stores base templates. | Runs a local engine that syncs its own cluster folder (like `clusters/<spoke-name>/`). | **Only path or branch overrides.** The main repo settings remain the same. |

In all cases, the GitOps engine belongs to the **hub**. Placing `gitops` inside each cluster was a mistake from early single-cluster versions.

---

### 2. Comparing the two options

#### Top-level `gitops:` (selected)
- **Clear single source.** Repository URLs and credentials are typed once.
- **Enforces the hub model.** Prevents conflicting engine choices in the same repository.
- **Simpler spoke entries.** Spoke clusters only list their stage, DNS, catalogs, and enabled services.
- **Handles overrides easily.** If a spoke needs a different branch, use a short `gitopsOverrides` field (like `targetRevision: staging`).

#### Per-cluster `clusters[].gitops` (old approach)
- **Too much repetition.** 20 spoke clusters mean repeating the same repository block 20 times.
- **Confusing.** Makes it look like every spoke runs its own GitOps server.
- **High risk of mistakes.** Different spoke entries can accidentally point to different repositories or engines.

---

## Target configuration schema (`config.yaml`)

### Key changes
- A top-level `gitops:` section holds all GitOps engine settings, repository links, credentials, and extra fleet tools.
- The `clusters[]` array no longer contains `argocd` blocks.
- Spoke clusters only specify cluster-specific information (stage, DNS, catalogs, enabled services).
- An optional `gitopsOverrides` field on a spoke cluster allows tracking a different git branch or revision when needed.

---

## Configuration examples

### 1. Standard hub with spokes (Argo CD push)

```yaml
# Top-level GitOps engine running on the hub cluster
gitops:
  engine: argocd
  selfManaged: enabled
  repo:
    https:
      configs:
        url: https://github.com/my-org/platform-configs.git
        targetRevision: main
      components:
        url: https://github.com/my-org/platform-components.git
        targetRevision: main
  helmRepo:
    url: https://charts.example.com

clusters:
  # Hub cluster: GitOps engine runs here
  - name: management-hub
    type: hub
    stage: prod
    dnsName: hub.example.com
    catalogs:
      - oci://ghcr.io/kubara-io/catalogs/general:2.3.0
    services:
      cert-manager:
        status: enabled
      traefik:
        status: enabled

  # Spoke cluster: Managed from the hub
  - name: workload-spoke-01
    type: spoke
    stage: prod
    dnsName: spoke01.example.com
    catalogs:
      - oci://ghcr.io/kubara-io/catalogs/general:2.3.0
    services:
      cert-manager:
        status: enabled
      traefik:
        status: enabled

  # Spoke cluster using a different git branch
  - name: workload-spoke-staging
    type: spoke
    stage: staging
    dnsName: spoke-stage.example.com
    gitopsOverrides:
      targetRevision: stage-branch
    catalogs:
      - oci://ghcr.io/kubara-io/catalogs/general:2.3.0
    services:
      cert-manager:
        status: enabled
```

---

### 2. Multi-cluster fleet (Sveltos add-ons + Argo CD Agent)

```yaml
gitops:
  engine: argocd
  selfManaged: enabled
  repo:
    https:
      configs:
        url: https://github.com/my-org/platform-configs.git
        targetRevision: main
      components:
        url: https://github.com/my-org/platform-components.git
        targetRevision: main
  flavors:
    # Sveltos matches cluster labels to install add-on profiles
    - name: sveltos
      config:
        syncMode: ContinuousWithDriftDetection
    # Agent connects edge spokes back to the hub
    - name: argocd-agent
      config:
        mode: managed

clusters:
  - name: management-hub
    type: hub
    stage: prod
    dnsName: hub.example.com
    catalogs:
      - oci://ghcr.io/kubara-io/catalogs/general:2.3.0
    services:
      cert-manager:
        status: enabled
      traefik:
        status: enabled

  - name: edge-spoke-factory-1
    type: spoke
    stage: prod
    dnsName: edge-f1.internal.example.com
    catalogs:
      - oci://ghcr.io/kubara-io/catalogs/general:2.3.0
    services:
      cert-manager:
        status: enabled
```

---

### 3. Pure Flux v2 setup

```yaml
gitops:
  engine: flux
  selfManaged: enabled
  repo:
    https:
      configs:
        url: https://github.com/my-org/platform-configs.git
        targetRevision: main
      components:
        url: https://github.com/my-org/platform-components.git
        targetRevision: main
  flavors:
    - name: sveltos
      config:
        syncMode: ContinuousWithDriftDetection

clusters:
  - name: flux-hub
    type: hub
    stage: prod
    dnsName: flux-hub.example.com
    catalogs:
      - oci://ghcr.io/kubara-io/catalogs/general:2.3.0
    services:
      cert-manager:
        status: enabled

  - name: flux-spoke-01
    type: spoke
    stage: dev
    dnsName: spoke01.dev.example.com
    catalogs:
      - oci://ghcr.io/kubara-io/catalogs/general:2.3.0
    services:
      cert-manager:
        status: enabled
```

---

## Outcomes

- **Good.** Top-level `gitops:` matches how kubara works: one control plane on the hub cluster.
- **Good.** Removes duplicate repository URLs and credentials across spokes.
- **Good.** Supports multiple engines and fleet flavors cleanly.
- **Good.** Keeps engine-specific logic out of Go code.
