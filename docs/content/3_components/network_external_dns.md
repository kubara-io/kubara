# Network: ExternalDNS

## What is ExternalDNS?

With kubara you can deploy **ExternalDNS** into your Kubernetes cluster (see https://kubernetes-sigs.github.io/external-dns/latest/).  
ExternalDNS ensures that DNS records are automatically created and updated as soon as you define Ingress, Service, or supported Traefik resources with hostnames.

> ExternalDNS will be rolled out as a **Helm Chart** when you enable the service in your `config.yaml`.

---

## Workflow

1. **Prepare DNS zone**  
   - kubara generates the necessary **Terraform definitions** (modules and variables) for DNS setup.  
   - You then run **Terraform** yourself to actually create the zone.  
   - If you use your own domain, you must delegate the nameservers of this zone at your registrar.  
   - Provider-specific behavior depends on your DNS platform.

2. **Enable ExternalDNS**  
   - In `config.yaml` you enable the service `externalDns`.  
   - Then you must **rerun kubara** (`kubara generate --terraform`, `kubara generate --helm` or `kubara generate`) so that Terraform files and Helm values are re-rendered with the new settings.  
   - Configure provider-specific values in:
     - `customer-service-catalog/helm/<cluster-name>/external-dns/values.yaml`
     - optional `customer-service-catalog/helm/<cluster-name>/external-dns/additional-values.yaml`
   - Next steps:  
     - run `terraform apply` to provision DNS resources,  
     - **git commit & push** the Helm chart changes so that ArgoCD/Flux deploys them to the cluster.  
   - At this point the ExternalDNS Helm Chart will be deployed from ArgoCD in the cluster.  

3. **Automatic records**  
   - When you deploy an application with Ingress or Service including a hostname (e.g. `app.example.com`), ExternalDNS automatically creates the corresponding DNS record in your configured zone.  
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
- **`terraform.dns`** → defines the zone for which kubara generates Terraform code (name and contact email).
- **`services.externalDns.status`** → when set to `enabled`, ExternalDNS is templated into the Helm charts for deployment via ArgoCD.
- **provider-specific settings** → configure them in the chart overlay values (`values.yaml` / `additional-values.yaml`).

---

## DNS Credentials

- DNS credentials are provider-specific and should be stored in your selected secret backend.
- The **External Secrets Operator** can sync those credentials into the cluster.
- The **ExternalDNS Helm Chart** consumes the synced secret for provider authentication.

Depending on your provider setup, credentials may be managed by Terraform, by your secret backend workflow, or manually.
