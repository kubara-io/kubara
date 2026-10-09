# Multi-Hub Environments

Managing multiple environments for example across geographical regions, different projects or stages is a common requirement.

Kubara supports multi-hub setups natively through **directory-scoped workspaces**. Each Hub cluster and its associated spokes live in an isolated directory, eliminating artifact collisions while automating GitOps repository path calculations.

---

## One-Hub-per-Config

Each hub folder represents a single hub-and-spoke management domain:

* Exactly **one** cluster with `type: hub` is configured per `config.yaml`.
* Any number of spoke clusters (`type: spoke`) can be managed by that hub.
* Different hubs require separate `config.yaml` files because each hub cluster represents an independent GitOps control plane with its own domain, credentials, and lifecycle.

---

## Repository Layout

Instead of having a single global `platform-components/` and `platform-configs/` directory at the repository root, each hub folder contains its own isolated set of components and configurations.

Hub directories can be at the top level or nested within subfolders. The root itself can also be a hub directory, like it is implicitly in a single-hub setup. 

```text
my-gitops-repo/
├── .git/
├── renovate.json                     # Shared Renovate config at repo root
├── prod/                             # Stage
│   ├── eu-west/                      # Prod Region 1 
│   │   ├── .env                      
│   │   ├── config.yaml               
│   │   ├── platform-components/      
│   │   └── platform-configs/
│   └── us-east/                      # Prod Region 2
│       ├── .env                      
│       ├── config.yaml               
│       ├── platform-components/      
│       └── platform-configs/         
│   
└── staging.  /                       # Another stage 
    ├── .env                         
    ├── config.yaml               
    ├── platform-components/      
    └── platform-configs/         
```


### Context Awareness

Kubara defaults to using the `config.yaml` and `.env` of the current working directory. 
When using the `--hubs` or `--all` flags it searches for hub folders below and relative to the current working directory. 
Running `kubara generate` inside of `my-gitops-repo/prod/eu-west/` produces the same result as running `kubara generate --hub prod/eu-west` inside of `my-gitops-repo`

```bash
# Generate configs for a single hub from repository root:
kubara generate --hub prod/us-east

# Target multiple hubs using comma-separated paths:
kubara generate --hub staging,prod/eu-west

# Target all hubs
kubara generate --all

```

### Template Context Exposure

Templates can access the computed workspace paths via the top-level `.workspace` object:

* `.workspace.gitRelativePath`: The relative path from the Git root (e.g., `"prod/eu-west"`, or `""` if at the repo root).
* `.workspace.platformComponents`: Path to components (`<gitRelativePath>/platform-components/helm`).
* `.workspace.platformConfigs`: Path to configs (`<gitRelativePath>/platform-configs`).

---

## Setting Up a New Hub 

Create the directory for your new hub and initialize it:

```bash
mkdir my-new-hub
cd my-new-hub
kubara init --prep
...
kubara init
...
kubara generate
```

The process is described in detail in [the Quick Start Guide](../1_getting_started/quick_start.md) and [Bootstrapping](../1_getting_started/bootstrapping.md). 


---

## Managing Spoke Clusters in Multi-Hub Setups

Adding a spoke cluster to a specific hub is straightforward. Simply run the command within that hub's workspace:

```bash
cd staging
kubara cluster add dev-spoke-1 --catalog /path/to/catalog
kubara generate
```

The spoke cluster's configuration is added to `staging/config.yaml`, and its generated overlays will live in `staging/platform-configs/dev-spoke-1/`. Its Argo CD applications will automatically point to the correct GitOps repository subpaths computed for the `staging` workspace.
