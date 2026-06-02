| status       | date       | decision-makers | consulted          | informed           |
|--------------|------------|-----------------|--------------------|--------------------|
| **accepted** | 2026-06-02 | kubara-Team     | Internal community | Internal community |

!!! note "Implementation status"
    Realized for the **T Cloud Public** path (OpenBao, kubernetes auth): every
    built-in consumer chart reads its secrets through a namespaced `SecretStore`
    scoped to `secret/<namespace>/*` via the `k8s-kv-read` role. The image pull
    secret remains on the cluster-wide store as the documented cross-namespace
    exception. **STACKIT is intentionally unchanged** and stays on the
    cluster-wide `ClusterSecretStore` until the userpass-vs-kubernetes-auth
    question below is resolved.

# Namespace-Isolated Secret Access via Per-Namespace SecretStores

## Context and Problem Statement

kubara provisions a single cluster-wide `ClusterSecretStore` backed by the configured secret manager (STACKIT Secrets Manager, or the in-cluster OpenBao deployed for the T Cloud Public CCE path). Every generated `ExternalSecret` references this one store. The store authenticates with a single fixed ServiceAccount and a backend role whose policy grants read access to the entire KV mount (`secret/data/*`). Platform secrets are stored under flat top-level keys such as `secret/docker_config`, `secret/grafana_credentials`, and `secret/velero_s3_credentials`.

The consequence is that the central External Secrets ServiceAccount can read **every** secret in the store, and any workload able to create an `ExternalSecret` (or a compromise of the External Secrets controller) inherits that reach. There is no per-namespace blast-radius containment.

The proven Vault + Vault Secrets Operator setup that predates kubara used per-namespace isolation instead: each pod authenticated from its own namespace's ServiceAccount, and a templated policy

```
path "secret/data/{{identity.entity.aliases.<accessor>.metadata.service_account_namespace}}/*"
```

scoped access to that namespace's subtree only. Secrets were organized as `secret/<namespace>/<name>`. The kubara OpenBao Terraform layer already ships this primitive (the `k8s-kv-read` role plus the templated policy), but nothing consumes it today because the `ClusterSecretStore` short-circuits it.

Should kubara move from the cluster-wide store with full-read access to per-namespace stores that enforce least-privilege isolation?

## Decision Drivers

* Least privilege and blast-radius containment: a namespace should only read its own secrets.
* Alignment with the proven legacy isolation model that teams already trust.
* Multi-tenant clusters where teams share a cluster but must not read each other's secrets.
* Auditability: which namespace may read which secret should be explicit, not implicit in one shared role.
* Keep secret generation declarative and provider-agnostic where possible.

## Considered Options

* Keep the cluster-wide `ClusterSecretStore` with full-read access (status quo)
* Per-namespace `SecretStore` objects with namespace-scoped KV paths and the templated `k8s-kv-read` policy
* Organizational-only namespacing: store secrets under `secret/<namespace>/` for tidiness but keep the cluster-wide store and full-read role

## Decision Outcome

Chosen option: **"Per-namespace `SecretStore` objects with namespace-scoped KV paths"**, because it is the only option that delivers real least-privilege isolation and it reuses the namespace-templated policy primitive kubara already generates. The organizational-only option was rejected because it changes nothing about who can read what (the central ServiceAccount still reads everything) and would give a false sense of isolation.

This is a breaking, cross-cutting change to how kubara consumes secrets across all providers, which is why it is captured as an ADR rather than implemented directly on a feature branch.

### Consequences

* **Good**, because each namespace's workloads can read only `secret/<namespace>/*`, matching least privilege and the legacy model.
* **Good**, because the existing `k8s-kv-read` role and templated policy finally carry real traffic instead of being dormant.
* **Good**, because a compromised workload or a stray `ExternalSecret` in one namespace cannot exfiltrate another namespace's secrets.
* **Bad**, because it is a breaking change to secret KV paths: existing clusters store secrets under flat keys and would need a migration (re-write KV entries under `secret/<namespace>/` and update every `ExternalSecret` remoteKey).
* **Bad**, because it multiplies Kubernetes objects: one `SecretStore` plus a bound ServiceAccount per consuming namespace, instead of one `ClusterSecretStore`.
* **Bad**, because every built-in chart's `externalSecrets` block and the shared external-secrets chart must change, affecting STACKIT and T Cloud Public alike.
* **Neutral**, because cluster-wide secrets that genuinely must be readable from many namespaces (for example the image pull secret distributed via `ClusterExternalSecret`) still need a deliberate exception rather than strict per-namespace scoping.

### Confirmation

A namespace's `SecretStore` ServiceAccount, when used to log in to the backend, must receive a token whose policy resolves to `secret/data/<that-namespace>/*` and must be denied reads of another namespace's path. This can be confirmed with a negative test: a `SecretStore` in namespace `a` attempting to read `secret/b/...` must fail. Generation-side, render tests assert that each chart emits a namespaced `SecretStore` (not a `ClusterSecretStore`) and that remoteKeys are prefixed with the consuming namespace.

---

## Pros and Cons of the Options

### Keep the cluster-wide ClusterSecretStore (status quo)

One `ClusterSecretStore`, one ServiceAccount, one full-read role; every `ExternalSecret` references it.

* **Good**, because it is simple: a single store object and a single backend role.
* **Good**, because it works uniformly across providers and auth methods (kubernetes auth for OpenBao, userpass for STACKIT Secrets Manager).
* **Bad**, because the central ServiceAccount can read every secret in the mount — no isolation.
* **Bad**, because it does not match the namespace-isolated model the organization already operates and trusts.

### Per-namespace SecretStores with namespace-scoped paths

Each consuming namespace gets a `SecretStore` that authenticates with a ServiceAccount in that namespace; the templated policy scopes access to `secret/<namespace>/*`.

* **Good**, because it enforces least privilege per namespace.
* **Good**, because it reuses the `k8s-kv-read` role and templated policy already generated.
* **Bad**, because it depends on **kubernetes auth**: the templated policy keys off `service_account_namespace`, which only exists for the Kubernetes auth method. The T Cloud Public OpenBao path uses kubernetes auth and supports this directly. The STACKIT Secrets Manager path currently uses **userpass** auth in the generated `ClusterSecretStore`, which carries no namespace identity — so STACKIT either needs a kubernetes-auth-capable configuration or cannot fully adopt this model. This provider asymmetry is the central open question of this ADR.
* **Bad**, because of the migration and object-count costs noted above.

### Organizational-only namespacing

Store secrets under `secret/<namespace>/` for readability in the backend UI, but keep the cluster-wide store and full-read role.

* **Good**, because it is a tiny change: only KV paths and remoteKeys move.
* **Bad**, because it provides **no** access isolation — the central ServiceAccount still reads everything. It looks isolated but is not, which is arguably worse than the honest flat layout.

## More Information

### Provider asymmetry (kubernetes auth vs userpass)

The isolation mechanism requires the backend auth method to expose the consuming namespace. With OpenBao kubernetes auth this is `service_account_namespace`; with STACKIT Secrets Manager userpass there is no equivalent. Realizing this ADR therefore means:

* **T Cloud Public (OpenBao, kubernetes auth):** fully supported today; the `k8s-kv-read` role and templated policy already exist.
* **STACKIT (Secrets Manager, userpass):** requires investigating whether STACKIT Secrets Manager can be driven with kubernetes auth or another per-namespace identity. If not, STACKIT may need to remain on the cluster-wide model, and this ADR would apply per provider rather than globally.

### Migration

Existing clusters store secrets under flat keys. A migration path (copy `secret/<key>` to `secret/<namespace>/<key>`, switch ExternalSecrets, then remove the flat keys) must be documented before this is rolled out to live clusters, and likely staged behind a feature flag during transition.

### Scope of follow-up work if accepted

* External-secrets chart: render per-namespace `SecretStore` objects (and the bound ServiceAccounts) instead of a single `ClusterSecretStore`.
* Every built-in chart with an `externalSecrets` block (argo-cd, kube-prometheus-stack, oauth2-proxy, external-dns, velero, …): switch `secretStoreRef` to the namespaced `SecretStore` and prefix remoteKeys with the namespace.
* OpenBao Terraform layer: write platform secrets under `secret/<namespace>/<name>`; keep the cross-namespace image pull secret as an explicit exception.
* Decide the STACKIT provider strategy (see provider asymmetry above).
