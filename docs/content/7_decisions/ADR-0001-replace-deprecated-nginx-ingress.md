| status       | date       | decision-makers  | consulted          | informed           |
|--------------|------------|------------------|--------------------|--------------------|
| **accepted** | 2026-01-23 | kubara-Team      | Internal community | Internal community |

# Replace Deprecated Nginx Ingress

## Context and Problem Statement

The `ingress-nginx` controller is deprecated; support ends in late March 2026 (see [ingress-nginx-retirement](https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/)). To prevent forced migrations for kubara users shortly after release, we need a stable, long-term replacement now.

## Decision Drivers

* **Maintainability:** Avoid self-built images or manual Helm chart maintenance.
* **Compatibility:** Stick to **Ingress API** for now to ensure compatibility with existing Helm charts and tools.
* **Ecosystem Readiness:** While Gateway API is stable as a specification, its implementation in upstream charts (ArgoCD, Prometheus, etc.) is still inconsistent or incomplete.
* **User Experience:** Must remain beginner-friendly.

## Considered Options

* **Traefik**
* **Chainguard (Nginx Fork)**
* **Envoy / Gateway API**
* **Istio**

## Decision Outcome

Chosen option: **"Traefik"**, because it acts as a robust "drop-in" replacement. It supports both Ingress and Gateway API in parallel, allowing us to remain on the stable Ingress API for compatibility while being future-proof. It offers a native dashboard and is significantly easier to manage than full service meshes or shifting entirely to the Gateway API ecosystem today.

### Consequences

* **Good:** Ease-of-use and nice features (e.g., dashboard).
* **Good:** Supports a hybrid approach (Ingress + Gateway API) for a gradual transition.
* **Bad:** Very small migration effort for users moving from Nginx-specific features (most nginx annotations are compatible with Traefik).

### Confirmation

Validation via staging deployments within the kubara stack, focusing on ingress routing and authentication (oauth2-proxy) compatibility.

---

## Pros and Cons of the Options

### Traefik

* **Good:** Parallel support for Ingress and Gateway API.
* **Good:** Beginner-friendly configuration and built-in dashboard.
* **Bad:** Minor effort to migrate Nginx-specific annotations.

### Chainguard (Nginx Fork)

* **Good:** Familiar technology (Nginx).
* **Bad:** Lack of free, pre-built public images and managed Helm charts makes it high-maintenance for the team.

### Envoy (Switching to Gateway API now)

* **Good:** High performance; native Gateway API focus.
* **Bad:** Low adoption in current third-party Helm charts (often still behind experimental flags).
* **Bad:** Steep learning curve and high migration overhead for end-users.

### Istio

* **Bad:** Significant resource overhead and complexity for a simple ingress replacement.
* **Bad:** Overkill for the current requirements of the kubara stack.

## More Information

We will continue to use the **Ingress API** as the primary interface. Although the Gateway API is functionally mature, it has not yet "arrived" everywhere in the Helm chart ecosystem. Traefik allows us to wait for full ecosystem adoption without being stuck on a deprecated controller.