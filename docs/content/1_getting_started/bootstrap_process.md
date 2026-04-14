# kubara: Bootstrapping Guide - ControlPlane Setup



## Introduction

This guide provides a step-by-step process for bootstrapping your platform running on Kubernetes, including the necessary [prerequisites](prerequisites.md), architecture setup, and deployment instructions. Try to follow the instructions first. If you have any questions or issues, please reach out directly via Teams. If you're interested in the setup details, explore the Wiki pages.

---

## 1. Getting Started

Whether you're running on STACKIT Cloud or STACKIT Edge, we recommend using the Terraform modules introduced in [Step 2](#2-terraform-provisioning-kubernetes-and-infrastructure-optional-recommended). If you already have a Kubernetes cluster without DNS, secrets management, etc., simply disable those services in the `config.yaml` file, which will be generated in the next steps.

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
    What is "type:" in `config.yaml`: `controlplane` is your hub cluster, `worker` is your spoke cluster [Hub and Spoke Cluster](../4_architecture/architecture_overview.md#hubnspoke)
!!! tip
    Not using STACKIT Edge? Just remove the load balancer IPs from your `config.yaml`.

Example:

```yaml
clusters:
  - name: project-name-from-env-file
    stage: project-stage-something-like-dev
    type: <controlplane or worker>
    dnsName: <cp.demo-42.stackit.run>
    privateLoadBalancerIP: 0.0.0.0
    publicLoadBalancerIP: 0.0.0.0
    ssoOrg: <oidc-org>
    ssoTeam: <org-team>
    terraform:
      provider: stackit # currently supported: stackit
      projectId: <project-id>
      kubernetesType: <ske or edge>
      kubernetesVersion: 1.34
      dns:
        name: <dns-name>
        email: <email>
...
```

`ingressClassName` defaults to `traefik`. Set it explicitly when using a different ingress controller.
Each service also accepts an optional `ingress.annotations` map under `services.<service>.ingress.annotations` that is merged with kubara's defaults, allowing you to add controller-specific annotations without overwriting the full set.
User-provided annotations are merged on top using `mergeOverwrite`: equal keys are overwritten, while kubara default keys that are not present in the override remain.

kubara generates resources in two stages:

* **Terraform modules and overlays** to provision infrastructure and the Kubernetes cluster
* **Helm templates** to deploy Argo CD and platform services

If you are not using Terraform, you can skip directly to Step 3.

---
## 2. Terraform: Provisioning Kubernetes and Infrastructure *(Optional, Recommended)*

!!! warning
    In kubara version `0.2.0`, this step does not merge user-customized Terraform values and will overwrite existing Terraform files.

Generate Terraform modules:

```bash
kubara generate --terraform
```

Commit and push the generated files to your Git repository.

!!! info 
    You will need access to the STACKIT API. Setup instructions are available in the [Terraform provider documentation](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs) & [STACKIT Docs](https://docs.stackit.cloud/platform/access-and-identity/service-accounts/how-tos/manage-service-accounts/).
    Make sure your created Service Account has Project Owner permissions.

### 2.1 Terraform Bootstrap

Before the first `terraform init`, prepare and load your environment variables:

```bash
cd customer-service-catalog/terraform/<cluster-name>
cp set-env-changeme.sh set-env.sh
```
Set at least `STACKIT_SERVICE_ACCOUNT_KEY_PATH` in `set-env.sh` / `set-env.ps1` before sourcing.
```bash
source set-env.sh
# or for PowerShell
# cp set-env-changeme.ps1 set-env.ps1
# . .\set-env.ps1
```


Then navigate to:

```bash
cd bootstrap-tfstate-backend
```

Run:

```bash
terraform init
terraform plan
terraform apply
```

Use the output to configure Terraform backend credentials:

```bash
terraform output debug | grep -E "credential_access_key|credential_secret_access_key"
```

You can set `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` in `set-env.sh` / `set-env.ps1` and source the file again, or export them directly:

```bash
export AWS_ACCESS_KEY_ID="<credential_access_key from terraform output>"
export AWS_SECRET_ACCESS_KEY="<credential_secret_access_key from terraform output>"
```

### 2.2 Provisioning Infrastructure
!!! warning
    If you do not intend to use OAuth2 Proxy you can ignore some of the steps in the guide below pertaining to it, but might encounter some issues later.
    For more infos please look at our [FAQ](../6_reference/faq.md#what-happens-when-oauth2-proxy-is-disabled).

Then proceed to:

```bash
cd ../infrastructure
```

Run:

```bash
terraform init
terraform plan
```
Check the values generated in `env.auto.tfvars`, which is [automatically applied in your Terraform deployment.](https://developer.hashicorp.com/terraform/language/values/variables#assign-values-to-variables)

```bash
terraform apply
```

This creates the Kubernetes cluster and all required infrastructure.

Export your kubeconfig:

```bash
# change command accordingly to your needs. For example change the name of your kubeconfig, to not overwrite any files
terraform output -raw kubeconfig_raw > $HOME/.kube/kubara.yaml
```

Keep this `kubara.yaml` local and do not commit it to Git.

Review Terraform outputs:

```bash
terraform output
```

Use Terraform outputs to update values in `config.yaml` where needed (for example, on Edge: `privateLoadBalancerIP` and `publicLoadBalancerIP`).
Do **not** export Secrets Manager credentials into `.env`; these provider-specific `.env` variables were removed.

Sensitive output example:

```bash
terraform output vault_user_ro_password_b64
```

If you use OAuth2, create a GitHub application as shown [here](../2_managing_your_platform/add_sso.md).

If you want Terraform to create OAuth2-related Vault entries:

* Use `set-env.sh` / `set-env.ps1` for `TF_VAR_*` in `customer-service-catalog/terraform/<cluster-name>/`
* `TF_Var_image_pull_secret` will already be set by kubara with what is present in the .env
* In `customer-service-catalog/terraform/<cluster-name>/infrastructure`, copy `secrets.tf-example` to `secrets-2.tf` and adjust values if needed

Load the variables and apply:

```bash
cp secrets.tf-example secrets-2.tf
source ../set-env.sh
# or for PowerShell
# Copy-Item secrets.tf-example secrets-2.tf
. ..\set-env.ps1
terraform apply
```

!!! warning
     You need to set these environment variables again before re-applying Terraform if they are not persisted in your shell/session setup.



To clean up:

```bash
terraform state rm \
  vault_kv_secret_v2.image_pull_secret \
  vault_kv_secret_v2.oauth2_creds \
  vault_kv_secret_v2.argo_oauth2_creds \
  vault_kv_secret_v2.grafana_oauth2_creds \
  random_password.oauth2_cookie_secret
```

### 2.3 STACKIT Edge-Specific Notes

The provisioning steps remain the same. The only difference lies in the Terraform output:

* You'll retrieve additional values like `privateLoadBalancerIP` and `publicLoadBalancerIP`
* These need to be added to `config.yaml`

You must manually create the Kubernetes cluster via the cloud portal. This will be automated in the future.

Now continue with Step 3.

---

## 3. Helm

This step extends the service catalog:

* Generates an umbrella Helm chart in `managed-service-catalog/`
* Creates cluster-specific overlays in `customer-service-catalog/`

```bash
kubara generate --helm
```


There are several Helm chart `values.yaml` files with dummy `change-me` values that need to be adjusted.
Example:
```yaml
# ... previous content of yaml file
admin: change-me
# ... rest of yaml
```
Edit the generated files in:
`customer-service-catalog/helm/<cluster>/<chart>/values.yaml`

Source templates are embedded in the binary under `go-binary/templates/embedded/...`, but you should only edit generated files in your repository.

The chart directories where values usually need review are:

* argo-cd
* cert-manager
* external-dns
* external-secrets
* homer-dashboard
* traefik
* kube-prometheus-stack
* kyverno-policy-reporter
* kyverno
* loki
* longhorn
* metallb
* metrics-server
* oauth2-proxy

### 3.1 Additional value files and CI value files

Every generated app supports:

* `values.yaml` as the main customer overlay file
* optional `additional-values.yaml` for overrides/extra values

kubara's generated ApplicationSet already references both files and ignores missing files, so you can add `additional-values.yaml` only when needed.

Merge behavior reminder:

* maps/dictionaries are merged recursively
* lists/arrays replace previous values completely

CI-specific values can be stored in chart-local CI files (for example `ci/ci-values.yaml`) to keep pipeline-only settings out of runtime overlays.


!!! warning
    **Don't forget to commit and push your changes to Git!**

---

## 4. Deploying Argo CD

### 4.1 Bootstrap the Control Plane

!!! warning
    This command requires access to a Kubernetes cluster and, by default, uses `~/.kube/config`.
    To target a specific cluster, provide your own config with `--kubeconfig your-kubeconfig`

For external-secrets, create provider credential secret(s) first (for example via `kubectl create secret ...`), then:

A) **recommended for first bootstrap:** pass a `ClusterSecretStore` manifest to bootstrap with `--with-es-css-file` together with `--with-es-crds`, 

or B) apply your `ClusterSecretStore` manually (only if external-secrets CRDs are already installed on the cluster).

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

```bash
kubara bootstrap <cluster-name-from-config-yaml> --with-es-crds --with-prometheus-crds
```

Recommended for the first bootstrap with external-secrets: let kubara apply a templated `ClusterSecretStore` manifest during bootstrap:

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

### Bootstrapping Multiple ControlPlanes

You can bootstrap multiple ControlPlanes.
We recommend **not** to reuse the same `config.yaml` file for multiple ControlPlanes.

**Why?**
During the bootstrap process, the `.env` file is used to provide credentials.
If you reuse the same `.env` file, you would have to constantly adjust it for each ControlPlane — which is error-prone.

Since version `0.2.0`, this is much easier. You can simply provide a different env file:

```bash
./kubara init --prep --env-file .env2
```
Fill out `.env2` with the required values. Generate a new config file from it:

```bash
./kubara --config-file config2.yaml --env-file .env2 init
```

This will use the values from `.env2` to generate `config2.yaml`.

Render Terraform modules and Helm charts for the new ControlPlane:

```bash
# default: generates both Helm and Terraform
# use --helm or --terraform to generate only one type
./kubara --config-file config2.yaml generate
```

Finally, bootstrap your additional ControlPlane:

```bash
./kubara --env-file .env2 bootstrap <cluster-name-from-config2-yaml> --with-es-crds --with-prometheus-crds
```

## What's Next?

After bootstrapping your platform, you can:

* [Add Argo CD projects](../2_managing_your_platform/add_app_project.md)
* [Add Git repositories](../2_managing_your_platform/add_app_repository.md)
* [Add Argo CD applications](../2_managing_your_platform/add_application.md)
* [Add Argo CD appset](../2_managing_your_platform/add_appset.md)
* [Add SSO Configuration](../2_managing_your_platform/add_sso.md)
* [Add additional worker clusters](../2_managing_your_platform/add_worker_cluster.md)
