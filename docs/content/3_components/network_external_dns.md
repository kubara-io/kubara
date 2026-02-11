# Network: ExternalDNS

## What is ExternalDNS?

With Kubara you can deploy **ExternalDNS** into your Kubernetes cluster (see https://kubernetes-sigs.github.io/external-dns/latest/).  
ExternalDNS ensures that DNS records are automatically created and updated as soon as you define Ingress or Service resources with hostnames.

> ExternalDNS will be rolled out as a **Helm Chart** when you enable the service in your `config.yaml`.

---

## Workflow

1. **Prepare DNS zone**  
   - Kubara generates the necessary **Terraform definitions** (modules and variables) for a DNS zone in STACKIT.  
   - You then run **Terraform** yourself to actually create the zone.  
   - If you use your own domain, you must delegate the nameservers of this zone at your registrar.  
   - If you use a STACKIT subdomain (for example `.runs.onstackit.cloud`), delegation is already in place - nothing else to do.

2. **Enable ExternalDNS**  
   - In `config.yaml` you enable the service `externalDns`.  
   - Then you must **rerun Kubara** (`kubara generate --terraform`, `kubara generate --helm` or `kubara generate`) so that Terraform files and Helm values are re-rendered with the new settings.  
   - Next steps:  
     - run `terraform apply` to provision the DNS zone in STACKIT,  
     - **git commit & push** the Helm chart changes so that ArgoCD/Flux deploys them to the cluster.  
   - At this point the ExternalDNS Helm Chart will be deployed from ArgoCD in the cluster.  

3. **Automatic records**  
   - When you deploy an application with Ingress or Service including a hostname (e.g. `app.example.com`), ExternalDNS automatically creates the corresponding DNS record in the STACKIT zone.  
   - Changes or deletions are also reflected automatically.

---

## Configuration in `config.yaml`

### Example

```yaml
clusters:
  - name: my-cluster
    stage: prod

    # Base domain / zone
    dnsName: example.com

    terraform:
      dns:
        name: "example-zone"
        email: "hostmaster@example.com"

    services:
      externalDns:
        status: enabled        # <--- enable here
```

### Explanation

- **`dnsName`** → base domain for the cluster  
- **`terraform.dns`** → defines the zone for which Kubara generates Terraform code (name and contact email).
- **`services.externalDns.status`** → when set to `enabled`, ExternalDNS is templated into the Helm charts for deployment via ArgoCD.
- **`services.externalDns.config`** → optional provider-specific settings (for STACKIT: webhook integration)

---

## DNS Credentials

- Kubara generates Terraform code that creates a **DNS admin (Vault KV)** entry in the **STACKIT Secrets Manager**.  
- The **External Secrets Operator** automatically syncs this secret into the Kubernetes cluster.  
- The **ExternalDNS Helm Chart** consumes this secret to authenticate against the STACKIT DNS provider.  

This means you do not need to manually create credentials inside the cluster - everything is handled securely via the Secrets Manager.
