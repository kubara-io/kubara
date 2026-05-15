# Security & Stability Best Practices

The short version: kubara is opinionated about **GitOps, version pinning, policy-based guardrails, observability, and hardened defaults for many bundled components**. It does **not** automatically make every workload in your cluster compliant or fully locked down.

The underlying best practices come from a mix of **official upstream Kubernetes and chart recommendations**, the **Kyverno policy ecosystem**, **IT-Grundschutz** (German Government) controls used in the shipped policy set, and **STACKIT / Schwarz Group operational experience** building and running Kubernetes platforms at scale.

---

## What kubara guarantees in its generated platform setup

### 1. GitOps-managed desired state

kubara generates declarative artifacts for platform components and is designed to have them reconciled by **Argo CD** from Git.

That gives you a few concrete guarantees:

- platform state is defined in files instead of manual cluster edits
- changes are reviewable and auditable in Git
- drift is detected by the GitOps controller instead of being silently accepted

### 2. Pinned component versions

kubara does not rely on floating `latest` chart dependencies for its built-in platform components. The managed catalog pins explicit chart versions, which improves reproducibility and reduces surprise upgrades.

### 3. Opinionated default service set

In the built-in catalog, several platform services are enabled by default for hub/spoke clusters, including:

- **cert-manager**
- **external-secrets**
- **external-dns**
- **traefik**
- **kyverno**
- **kyverno-policies**
- **kube-prometheus-stack**
- **loki**

This means a standard kubara setup starts with policy, observability, TLS automation, and secret integration already part of the platform design instead of being optional afterthoughts.

---

## Security practices kubara uses by default

### Policy engine and policy pack are enabled by default

kubara enables **Kyverno** and its bundled **kyverno-policies** chart by default. The shipped policy set is **audit-first**: it surfaces violations early without immediately blocking every non-conforming workload.

Out of the box, the policy pack covers areas such as:

- Pod Security Standard profile selection (`baseline`)
- disallowing privileged containers
- disallowing privilege escalation
- requiring probes
- requiring CPU and memory requests/limits
- requiring read-only root filesystems
- requiring PodDisruptionBudgets
- disallowing use of the `default` namespace
- disallowing the `latest` image tag

Several of these policies are aligned with **IT-Grundschutz-style** controls and are packaged directly into the managed catalog.

### Hardened runtime settings for many bundled components

Where the upstream chart supports it, kubara sets security-focused container defaults such as:

- `runAsNonRoot: true`
- `allowPrivilegeEscalation: false`
- `readOnlyRootFilesystem: true`
- `seccompProfile.type: RuntimeDefault`
- dropping Linux capabilities with `drop: [ALL]`

You can already see these defaults in bundled components such as **ExternalDNS**, **Loki**, and large parts of **kube-prometheus-stack**.

### Secret handling prefers external backends

kubara ships **External Secrets Operator** as a default component and built-in charts consume Kubernetes `Secret` references for sensitive values instead of hardcoding credentials into manifests where possible.

That does not prevent you from creating plain Kubernetes `Secret` objects yourself, but the default platform path is to **sync secrets from an external backend**.

### TLS automation is part of the default platform story

kubara enables **cert-manager** by default and generates a `ClusterIssuer` configuration path for automated certificate issuance and renewal.

The bundled Kyverno policy set also includes audit rules for certificate-related controls such as:

- allowed DNS names
- certificate duration limits
- issuer restrictions

---

## Stability practices kubara uses by default

### Resource requests and limits are set for bundled platform components

The managed catalog sets resource requests and limits for many shipped components. This improves scheduling predictability and reduces the chance that essential platform services compete without guardrails.

### Probes and availability checks are part of the default posture

kubara enables liveness/readiness probes for several bundled components and also ships a Kyverno policy that audits workloads missing health probes.

### Monitoring and alerting are treated as platform primitives

The default service catalog enables **kube-prometheus-stack** and **Loki**, so metrics, alerts, and logs are part of the baseline platform architecture.

kubara also ships Argo CD alerting for cases such as:

- applications not being synchronized for a long time
- applications being unhealthy
- Argo CD not reporting applications at all

### Some noise-reduction defaults are applied for managed clusters

The Prometheus stack disables scraping/alerting for control-plane components that are commonly unreachable in managed Kubernetes offerings. This avoids permanent false-positive alerts for components like `kubeScheduler`, `kubeControllerManager`, `kubeProxy`, and `kubeEtcd`.

---

## Available in kubara, but **not** guaranteed by default

Some stronger controls exist in the catalog, but kubara does not enable them automatically:

- automatic creation of **default-deny NetworkPolicies** for existing namespaces
- automatic creation of **default-deny NetworkPolicies** for new namespaces
- **image registry allow-listing**
- **image signature verification**

These are available as policy options, but they are intentionally off by default because they can be disruptive in existing environments.

---

## Important limits and caveats

kubara is opinionated, but it is not magic. A few important boundaries:

- **Policies are audit-first by default.** They improve visibility, but they do not fully enforce compliance until you switch selected rules to `Enforce`.
- **Not every workload is hardened automatically.** kubara hardens many bundled platform components, but your own application charts still need to meet your security baseline.
- **Some upstream components need explicit exceptions.** The shipped Kyverno configuration contains exclusions for system workloads and components that legitimately need different settings.

---

## Practical interpretation

If you use kubara as intended, you get a platform that is:

- **GitOps-driven and reproducible**
- **version-pinned**
- **observable by default**
- **guarded by policy by default**
- **already hardened in many of its built-in components**

If you need **strict enforcement**, **cluster-wide default-deny networking**, or **image verification**, kubara gives you a starting point, but you still need to turn those controls on and validate them in your own environment.
