# Security: Certificate Management

## What is Certificate Management?

With kubara you can automatically manage **TLS certificates** in your Kubernetes cluster using **cert-manager** (see https://cert-manager.io/).  
cert-manager handles requesting, renewing, and storing certificates so that your applications are always served with valid TLS.

> cert-manager will be rolled out as a **Helm Chart** via Argo CD when you enable the service in your `config.yaml`.

---

## Workflow

1. **Requirement: working domain**  
   - In order for a certificate provider to issue certificates, your domain must correctly resolve to the Ingress controller in the cluster.  
   - When using ACME providers (e.g. Let's Encrypt), validation usually happens via **http-01 challenges**: the provider calls a special path (`/.well-known/acme-challenge/...`) on your domain.  
   - Whether you manage DNS records manually or automate them with [ExternalDNS](../3_components/network_external_dns.md) does not matter - what matters is that the domain resolves correctly.

2. **Enable cert-manager**  
   - In `config.yaml` you enable the service `certManager`.  
   - Then you must **rerun kubara** (`kubara generate --helm` or `kubara generate`) so that Helm values are re-rendered with the new settings.  
   - Then **git commit & push** the Helm chart changes so that Argo CD deploys cert-manager to the cluster.

3. **Configure ClusterIssuer**  
   - kubara automatically renders a **ClusterIssuer** that you can configure in `config.yaml`.  
   - Typical parameters:  
     - **Email address** (required for ACME providers)  
     - **Server URL** of the ACME provider (e.g. Let's Encrypt production or staging, or another ACME provider)  
   - This allows you to choose any certificate provider you want to use.

4. **Automatic certificates**  
   - When you deploy an Ingress annotated with `cert-manager.io/cluster-issuer: <issuer-name>`, cert-manager will automatically request a TLS certificate.  
   - The provider validates the domain (e.g. via http-01 challenge).  
   - After successful validation the certificate is issued and stored as a Kubernetes Secret.  
   - Ingress resources then automatically mount this Secret.  
   - cert-manager renews certificates automatically before they expire.

---

## Configuration in `config.yaml`

### Example

```yaml
clusters:
  - name: my-cluster
    stage: prod

    dnsName: example.com

    services:
      certManager:
        status: enabled
        config:
          email: "admin@example.com"    # Email for ACME
          server: "https://acme-v02.api.letsencrypt.org/directory"   # ACME URL (e.g. Let's Encrypt)
```

### Explanation

- **`services.certManager.status`** → when set to `enabled`, cert-manager is templated into the Helm charts for deployment via Argo CD
- **`services.certManager.config.email`** → email address registered with the ACME provider (required)  
- **`services.certManager.config.server`** → endpoint of the ACME provider (e.g. Let's Encrypt staging/production or another ACME provider)  

---

## Certificate issuance flow

- cert-manager sends a request to the configured provider.  
- The provider validates the domain (e.g. via http-01 challenge).  
- After successful validation the certificate is issued.  
- cert-manager stores the certificate as a Kubernetes Secret.  
- Ingress resources reference this Secret automatically.  
- cert-manager renews certificates on time without manual action.

---

## Note on DNS / ExternalDNS

- [ExternalDNS](../3_components/network_external_dns.md)  is **not strictly required**.  
- You can manage DNS records manually as long as the domain resolves to the cluster.  
- With [ExternalDNS](../3_components/network_external_dns.md)  it becomes easier, since records are created and updated automatically.  
