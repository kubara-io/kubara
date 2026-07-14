# Bootstrap Your Own Platform

## Introduction

This guide provides a step-by-step process for bootstrapping your platform running on Kubernetes, including the necessary [prerequisites](prerequisites.md), architecture setup, and deployment instructions. Try to follow the instructions first. If you have any questions or issues, please reach out directly via Teams. If you're interested in the setup details, explore the Wiki pages.

---

## 1. Getting Started

kubara can run on any Kubernetes cluster as long as the required surrounding capabilities exist, especially a secret backend for `external-secrets` and DNS handling for `external-dns`.

For the ready-made example flows, use the [Infrastructure Presets](../3_infrastructure/overview.md) section. Whether you're running on STACKIT Cloud, STACKIT Edge, T Cloud Public, 
or other cloud providers, we recommend you to use IaC (Infrastructure-as-Code) be it Terraform/Tofu, Pulumi or other solutions.

If you already have a Kubernetes cluster without DNS, secrets management, etc., simply disable those services in the `config.yaml` file, which will be generated in the next steps.

### 1.1 Environment Configuration

Refer to the [Prerequisites](prerequisites.md) guide and ensure all non-optional tasks in that guide are completed.<br>
Don't forget to create a new Git repository - all following steps should be executed from within that newly created repository.<br>
The easiest way is to run `kubara` inside the repository (but do not add the binary to Git).

---

### 1.2 Generate preparation files

1. Run this command to scaffold essential setup files:

    ```bash
    kubara init --prep
    ```

    This will generate:

    * A `.gitignore` file to help prevent accidental commits of sensitive or unnecessary files
    * An `.env` file that serves as a template for your environment configuration.
      Fill all placeholders (`<...>`) before running `kubara init`.

    !!! tip "Working with coding agents"
        Run `kubara agents` to scaffold an `AGENTS.md` into your repository so tools like Claude
        Code or Codex get a compact, token-lean entry point instead of crawling the full docs site.
        It points at `kubara --help` and `kubara schema` as the source of truth and links the
        **raw Markdown** documentation pinned to your installed kubara version. Commit it so it
        travels with the repository, and re-run `kubara agents --overwrite` to refresh it after
        upgrading kubara.

2. Update the values inside `.env`

    !!! danger "Handling .env Files"
        `.env` files contain sensitive credentials and must be treated as secrets.
        Never commit a plain `.env` file directly into Git.
        If you really need it in the repository, make sure it is stored in encrypted form only.
        Always add `.env` to `.gitignore` to avoid accidental commits.
        For team collaboration, proven approaches include encrypted `.env` files in the repository, centralized secret management, or helper tools like `dotenv`.
        Important: A plain `.env` file in Git exposes all secrets and must be avoided.

3. Check your values

    !!! warning
        Keep in mind that weak passwords such as `123456` for `ARGOCD_WIZARD_ACCOUNT_PASSWORD` are a bad idea, since your
        platform will be publicly available by default via your DNS zone.



### 1.3 Generate Base Configuration

Initialize your configuration:

```bash
kubara init
```

This command creates a `config.yaml` file based on the values from your `.env`.
If you make changes to `.env` later, you can re-run the command with `--overwrite` to update the configuration.

When using `--overwrite`, only values from `.env` are replaced.
Additional settings in your existing `config.yaml` are preserved and merged.
This currently applies **only to the first cluster entry**.

!!! info 
    If you plan to use Velero here, and have velero enabled, you need to also put in the s3Url inside the service definition file. More info about this [here](../6_components/backup_and_recovery.md).

### 1.4 Validate your `config.yaml` against schema (optional, recommended)

Generate a JSON schema file:

```bash
kubara schema
```

By default this creates `config.schema.json` in your current directory.
You can set a custom output file:

```bash
kubara schema --output custom-config.schema.json
```

For editor integration (e.g. VS Code with YAML language server), reference the schema in your `config.yaml`:

```yaml
# yaml-language-server: $schema=./config.schema.json
```

### 1.5 Update and Prepare Templates

!!! info
    What is "type:" in `config.yaml`: `hub` is your hub cluster, `spoke` is your spoke cluster [Hub and Spoke Cluster](../7_architecture/architecture_overview.md#hubnspoke)

!!! tip
    **Not using STACKIT Edge?** Just remove the load balancer IPs from your `config.yaml`.  
    **Not using an SSO provider?** Just set the ssoOrg and ssoTeam to "none"


Example:

```yaml
clusters:
  - name: project-name-from-env-file
    stage: project-stage-something-like-dev
    type: <hub or spoke>
    dnsName: <cp.demo-42.stackit.run>
    # Hint: If you don't use an SSO provider, you can set ssoOrg and ssoTeam to "none"
    ssoOrg: <oidc-org> 
    ssoTeam: <org-team>
    terraform:
      provider: stackit # currently supported: stackit, t-cloud-public
      projectId: <project-id-or-tenant-name>
      kubernetesType: <ske, edge or cce>
      kubernetesVersion: 1.34
      dns:
        name: <dns-name>
        email: <email>
...
    services:  
      ...
      metallb:
        status: enabled
        config:
          loadBalancerAddressPool:
            - 0.0.0.0
          publicLoadBalancerIPs: 0.0.0.0
...
```

`terraform.projectId` is provider-specific. For `t-cloud-public`, use the T Cloud Public tenant/project name that the Terraform provider expects as `tenant_name`, not a UUID.

`ingressClassName` defaults to `traefik`. Set it explicitly when using a different ingress controller.
Each service also accepts an optional `ingress.annotations` map under `services.<service>.ingress.annotations` that is merged with kubara's defaults, allowing you to add controller-specific annotations without overwriting the full set.
User-provided annotations are merged on top using `mergeOverwrite`: equal keys are overwritten, while kubara default keys that are not present in the override remain.

kubara generates resources in two stages:

* **Terraform modules and overlays** to provision infrastructure and the Kubernetes cluster
* **Helm templates** to deploy Argo CD and platform services

If you are not using Terraform, you can skip directly to Step 3.

## 2. Infrastructure Provisioning

kubara is mainly focused on the platform layer on top of Kubernetes.
For some environments we provide ready-made infrastructure presets and generated Terraform examples.

Visit the [Infrastructure Presets](../3_infrastructure/overview.md) section for:

- [STACKIT SKE](../3_infrastructure/stackit_ske.md)
- [STACKIT Edge Cloud](../3_infrastructure/stackit_edge_cloud.md)
- [T Cloud Public](../3_infrastructure/t-cloud-public.md)
- More provider support is in the works

If you already have an existing Kubernetes cluster and a secret manager supported by `external-secrets`, continue with the next section.

---
## 3. Helm

This step extends the service catalog:

* Generates an umbrella Helm chart in `platform-components/`
* Creates cluster-specific overlays in `platform-configs/`

```bash
kubara generate --helm
```


The generated `values.generated.yaml` files are pre-filled from your `config.yaml` and `.env`. Review them if you 
want to understand more details about the predefined settings but DO NOT edit those files as they will be regenerated
with future updates. If you need customization you can add as many files to the charts with the pattern `values-*.yaml`.
Which will be merged in lexical order. Hint: Lists in YAML files cannot be merged by ArgoCD/Helm. They will be completely
overwritten by the last file that includes the list.

Source templates are embedded in the binary under `src/internal/catalog/built-in/...`, but you should only edit generated files in your repository.

The chart directories where values usually need review are:

* argo-cd
* cert-manager
* external-dns
* external-secrets
* homer-dashboard
* kube-prometheus-stack
* kyverno
* kyverno-policy-reporter
* loki
* longhorn
* metallb
* metrics-server
* oauth2-proxy
* traefik
* velero

### 3.1 Additional value files and CI value files

Every generated app supports:

* `values.generated.yaml` as the main generated overlay file
* optional `values-*.yaml` for overrides/extra values (merged in lexical order)

Merge behavior reminder:

* maps/dictionaries are merged recursively
* lists/arrays replace previous values completely

CI-specific values can be stored in chart-local CI files (for example `ci/ci-values.yaml`) to keep pipeline-only settings out of runtime overlays.

!!! info "T Cloud Public CCE: provider-specific Helm adjustments"
    After `kubara generate --helm` and before `kubara bootstrap`, replace the Traefik service annotation placeholder with the shared load balancer ID from Terraform:

    ```bash
    cd platform-configs/<cluster>/terraform/infrastructure
    terraform output -raw load_balancer_id
    ```

    Set the value in `platform-configs/<cluster>/helm/traefik/values.generated.yaml` under `traefik.service.annotations["kubernetes.io/elb.id"]`. Keep `kubernetes.io/elb.class: "union"`.

    ExternalDNS also needs a Kubernetes Secret named `tcloudpubliccloudsyaml` with a `clouds.yaml` key. With the default OpenBao and External Secrets setup, copy the ExternalDNS block from `platform-configs/<cluster>/terraform/openbao/secrets.tf-oauth2` to `secrets.tf`, set `TF_VAR_external_dns_os_username` and `TF_VAR_external_dns_os_password` in `set-env.sh`, and apply the OpenBao Terraform layer before bootstrap. If you need Terraform value overrides in the OpenBao layer, use [Terraform value overrides](../2_concepts/overview_core_concept.md#terraform-value-overrides).

!!! warning
    **Don't forget to commit and push your changes to Git!**

---

## 4. Deploying Argo CD

### 4.1 Bootstrap the Hub cluster

!!! warning
    This command requires access to a Kubernetes cluster and, by default, uses `~/.kube/config`.
    To target a specific cluster, provide your own config with `--kubeconfig your-kubeconfig`

External Secrets needs a `ClusterSecretStore`.

For T Cloud Public CCE, kubara already renders the OpenBao-backed `ClusterSecretStore` into `external-secrets/values.yaml`. Use `--with-es-crds`; do not pass `--with-es-css-file` and do not create a separate provider credential secret.

For other providers, create the provider credential secret first and either pass a `ClusterSecretStore` manifest during bootstrap with `--with-es-css-file` and `--with-es-crds`, or apply the `ClusterSecretStore` manually if the External Secrets CRDs already exist.

If the namespace does not exist yet, create it once before creating the provider credential secret(s):

```bash
kubectl create namespace external-secrets
```

Example provider credential secret(s) for the `ClusterSecretStore`:

```bash
## Bitwarden
kubectl -n external-secrets create secret generic bitwarden-access-token \
  --from-literal=token="<BITWARDEN_MACHINE_ACCOUNT_TOKEN>"

## STACKIT Secrets Manager
kubectl -n external-secrets create secret generic stackit-secrets-manager-cred \
  --from-literal=username="<USERNAME>" \
  --from-literal=password="<PASSWORD>"
```

Example `clustersecretstore.yaml` for `--with-es-css-file` (templating with `{{ .cluster.name }}` / `{{ .cluster.stage }}` is supported):

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  labels:
    argocd.argoproj.io/instance: {{ .cluster.name }}-external-secrets
  name: "{{ .cluster.name }}-{{ .cluster.stage }}"
spec:
  provider:
    vault:
      auth:
        userPass:
          path: userpass
          secretRef:
            key: password
            name: stackit-secrets-manager-cred
            namespace: external-secrets
          username: "<USERNAME>"
      path: "<VAULT_PATH>"
      server: "https://<your-secrets-manager-endpoint>"
      version: v2
```

kubara scopes secret paths by cluster and stage. Namespace-specific secrets use
`<cluster-name>/<stage>/<namespace>/<secret>`, while cluster-wide secrets use
`<cluster-name>/<stage>/cluster_secrets/<secret>`. For example, Grafana credentials for the
`controlplane` production cluster live at
`controlplane/production/kube-prometheus-stack/grafana_credentials`.
This path layout is the same for every provider and secret backend, including the local OpenBao setup.

```bash
kubara bootstrap <cluster-name-from-config-yaml> --with-es-crds --with-prometheus-crds
```

For providers that need an external `ClusterSecretStore` manifest, pass it during bootstrap:

```bash
kubara bootstrap <cluster-name-from-config-yaml> \
  --kubeconfig k8s.yaml \
  --with-es-css-file clustersecretstore.yaml \
  --with-es-crds --with-prometheus-crds
```

After a successful bootstrap run, your platform should be operational.

---

## 5. Access the Argo CD Dashboard

!!! info "Argocd Local Login"
    **Username:** `wizard`  
    **Password:** From `.env` (`ARGOCD_WIZARD_ACCOUNT_PASSWORD`)

1. Start port-forwarding:

   ```bash
   kubectl port-forward svc/argocd-server -n argocd 8080:443
   ```

2. Open your browser at: [http://localhost:8080/argocd](http://localhost:8080/argocd)

3. Log in with the credentials above.

Enjoy your new platform!

---

## What's also possible?

This section will be extended in the future to describe not just technical changes,
but also other supported possibilities when bootstrapping.

### Bootstrapping Multiple Hub Cluster

You can bootstrap multiple Hub clusters.
Do **not** reuse the same `config.yaml` file for multiple Hub clusters.

**Why?**
During the bootstrap process, the `.env` file is used to provide credentials.
If you reuse the same `.env` file, you would have to constantly adjust it for each Hub - which is error-prone.

Since version `0.2.0`, this is much easier. You can simply provide a different env file:

```bash
kubara init --prep --env-file .another-env
```
Fill out `.another-env` with the required values. Generate a new config file from it:

```bash
kubara --config-file another-config.yaml --env-file .another-env init
```

This will use the values from `.another-env` to generate `another-config.yaml`.

Render Terraform modules and Helm charts for the new Hub cluster:

```bash
# default: generates both Helm and Terraform
# use --helm or --terraform to generate only one type
./kubara --config-file another-config.yaml generate
```

Finally, bootstrap your additional Hub cluster:

```bash
kubara bootstrap --config-file another-config.yaml --env-file .another-env <cluster name from another-config.yaml> --with-es-crds --with-prometheus-crds
```

## What's Next?

After bootstrapping your platform, you can:

* [How to add Argo CD projects](../5_workload_onboarding/add_app_project.md)
* [How to add Git repositories](../5_workload_onboarding/add_app_repository.md)
* [How to add Argo CD applications](../5_workload_onboarding/add_application.md)
* [How to add Argo CD appset](../5_workload_onboarding/add_appset.md)
* [How to add SSO Configuration](../4_building_your_platform/sso/add_sso.md)
* [How to add spoke clusters](../4_building_your_platform/add_spoke_cluster.md)
