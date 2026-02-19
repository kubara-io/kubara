# Kubara: Bootstrapping Guide - ControlPlane Setup



## Introduction

This guide provides a step-by-step process for bootstrapping your platform running on Kubernetes, including the necessary [prerequisites](prerequisites.md), architecture setup, and deployment instructions. Try to follow the instructions first. If you have any questions or issues, please reach out directly via Teams. If you're interested in the setup details, explore the Wiki pages.

---

## 1. Getting Started

Whether you're running on STACKIT Cloud or STACKIT Edge, we recommend using the Terraform modules introduced in [Step 2](#2-terraform-provisioning-kubernetes-and-infrastructure-optional-recommended). If you already have a Kubernetes cluster without DNS, secrets management, etc., simply disable those services in the `config.yaml` file, which will be generated in the next steps.

### 1.1 Environment Configuration

Refer to the [Prerequisites](prerequisites.md) guide and ensure all non-optional tasks in that guide are completed.<br>
Don't forget to create a new git repository - all following steps should be executed from within that newly created repository.<br>
The easiest way is to run `kubara` inside the repository (but do not add the binary to git).

---

### 1.2 Generate preparation files

1. Run this command to scaffold essential setup files:

    ```bash
     kubara init --prep
    ```

   This will generate:

    * A `.gitignore` file to help prevent accidental commits of sensitive or unnecessary files
    * An `.env` file that serves as a template for your environment configuration, adjust the values as needed.
      > ⚠️ If you are not using Edge or adding a worker cluster, provide dummy values:
      use a valid IPv4 placeholder such as 0.0.0.0, and for all WORKER_... variables just set `dummy` strings.

2. Update the values inside `.env`

   > **⚠️ Handling .env Files**
   .env files contain sensitive credentials and must be treated as secrets.
   Never commit a plain .env file directly into Git.
   If you really need it in the repository, make sure it is stored in encrypted form only.
   Always add `.env` to `.gitignore` to avoid accidental commits.
   For team collaboration there are several proven approaches e.g.: encrypted .env files in the repository / centralized management with a secret manager / helper tools like `dotenv`.
   Important: A plain .env file in Git exposes all secrets and must be avoided.

3. Check your values
   > ⚠️ Keep in mind that Passwords like "123456" as "ARGOCD_WIZARD_ACCOUNT_PASSWORD" wouldn't be a good idea since your 
   > Platform will be publicly available by default via your DNS Zone.



### 1.3 Generate Base Configuration

Initialize your configuration:

```bash
kubara init
```

This command creates a `config.yaml` file based on the values from your `.env`.
If you make changes to the .env later, you can re-run the command with `--overwrite` to update the configuration.

When using `--overwrite`, only the values from `.env` are replaced, while any additional settings in your existing `config.yaml` are preserved and merged, but this applies **only to the first cluster entry**

### 1.4 Update and Prepare Templates

> ⚠️ This step includes a typical chicken-and-egg scenario. You'll first need to provision essential resources like Vault. Only then can you extract required values such as `externalSecrets.path`, `externalSecrets.userName`, `privateLoadBalancerIP`, and `publicLoadBalancerIP` to configure your `config.yaml`. If some values are missing just use dummy values and replace them later after you have all values.

> 💡 What is "type:" in `config.yaml`: Controlplane-Cluster is a synonym for Hub-Cluster. Worker for Spoke-Cluster [Hub and Spoke Cluster](../4_architecture/architecture_overview.md#hubnspoke)
> 💡 Not using STACKIT Edge? Just remove the load balancer IPs from your `config.yaml`.

Example:

```yaml
clusters:
  - name: project-name-from-env-file
    stage: project-stage-something-like-dev
    projectId: project-ID-from-env-file
    type: <controlplane or workerplane> 
    dnsName: <cp.demo-42.stackit.run>
    privateLoadBalancerIP: 0.0.0.0
    publicLoadbalancerIP: 0.0.0.0
    ssoOrg: <oidc-org>
    ssoTeam: <org-team>
    terraform:
      dns:
        kubernetesType: <ske or edge>
        name: <dns-name>
        email: <email>
...
```

Kubara templates resources in two stages:

* **Terraform modules and overlays** to provision infrastructure and the Kubernetes cluster
* **Helm templates** to deploy Argo CD and platform services

If you are not using Terraform, you can skip directly to step 3.

---
## 2. Terraform: Provisioning Kubernetes and Infrastructure *(Optional, Recommended)*

> ⚠️  This step doesnt allows in the kubara version `0.2.0` to merge terraform custom values made by the user and will overwrite the existing terraform files.

Generate Terraform modules:

```bash
kubara generate --terraform
```

Commit and push the generated files to your Git repository.

### 2.1. Terraform Bootstrap
> 📘 You will need access to the STACKIT API. Setup instructions are available in the [Terraform provider documentation](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs).<br>
> More information is available under [STACKIT Service Account](https://docs.stackit.cloud/platform/access-and-identity/service-accounts/)

After you've successfully created a STACKIT Service Account, you should have a .json with public and secret key inside of it. Put the absolute path of this .json into the 'set-env-changeme.sh' and source it.
`source <path>/set-env-changeme.sh`
The set-env-changeme.sh can be found inside the `customer-service-catalog/terraform/<cluster-name>/terraform` folder.
This file stays on your computer and is included in the .gitignore. It is important that you keep your set-env-changeme.sh local and not accidentaly check it into the source versioning, as it contains secrets.

Having sourced the set-env-change.me.sh now navigate to:
```bash
cd customer-service-catalog/terraform/<cluster-name>/bootstrap-tfstate-backend
```
Inside the directory run:
```bash
terraform init
terraform plan
terraform apply
```
After the programs have run successfully run this command: 
```bash
terraform output debug
```
Use the output to configure Terraform backend credentials:
Set the files in the set-env-changeme.sh or .ps1 and source them. 
Or you can export them manually via:
```bash
export AWS_ACCESS_KEY_ID="<from terraform output>"
export AWS_SECRET_ACCESS_KEY="<from terraform output>"
```

### 2.2. Provisioning Infrastructure

Then change the directory to:
```bash
cd customer-service-catalog/terraform/<cluster-name>/infrastructure
```

Run:
```bash
terraform init
terraform plan
```
Check the values generated in env.auto.tfvars which is [automatically applied in your Terraform-Deployment.](https://developer.hashicorp.com/terraform/language/values/variables#assign-values-to-variables)

The following command creates the Kubernetes cluster and all required infrastructure.
```bash
terraform apply
```

Export your kubeconfig:

```bash
terraform output -json kubeconfig_raw | jq -r > k8s.yaml
```
For easier reference, put this k8s.yaml inside your project root directory.
This `k8s.yaml` also should **NOT** be included in your git.


Now having generated the config for kubernetes, we are ready for the setting up the .env for the second part.
Inside the infrastructure directory run.
```bash
terraform output
```
You can get \<sensitive\> values (e.g passwords) by executing:
```bash
terraform output vault_user_ro_password_b64
```
Copy and paste the variables like `vault_user_ro_name` from the output into the to `.env` and `config.yaml`:
In the .env the `SECRETS_MANAGER_PATH` is the `vault_instance_id` and the `SECRETS_MANAGER_URL` is the API endpoint for your secrets manager provider.

> **Note:** Secrets for OAuth2 and Argo CD SSO are not created automatically. Use the provided `secrets.tf-<clustername>` file or insert them manually into Vault.

If you use OAuth2, create a GitHub application as shown [here](../2_managing_your_platform/add_sso.md). We're transitioning to STACKIT Managed Git based on Forgejo.

Set the OAuth2 environment variables in the set-env-change.sh and after having done that source them again using:

```bash
# Assuming you're still in the <path>/terraform/<project-name>/infrastructure folder
source ../set-env-changeme.sh
# or for PowerShell
. .\set-env-changeme.ps1
```
Then still inside the infrastructure directory, 
rename `secrets.tf-<clustername>` to `secrets-2.tf` and apply:

```bash
terraform apply
```

> ⚠️ You will need to set these environment variables every time you re-apply Terraform. This is only required during bootstrapping.

To clean up:

```bash
terraform state rm \
  vault_kv_secret_v2.image_pull_secret \
  vault_kv_secret_v2.oauth2_creds \
  vault_kv_secret_v2.argo_oauth2_creds \
  vault_kv_secret_v2.grafana_oauth2_creds \
  random_password.oauth2_cookie_secret
```

#### Finally:
Commit the files to your repo again. Pay attention not to commit secrets (e.g no k8s.yaml, secrets-*.tf, .env etc.)

### 2.4. STACKIT Edge-Specific Notes

The provisioning steps remain the same. The only difference lies in the Terraform output:

* You'll retrieve additional values like `privateLoadBalancerIP` and `publicLoadBalancerIP`
* These need to be added to `config.yaml`

You must manually create the Kubernetes cluster via the IEP/SIT cloud portal. This will be automated in the future.

We can now continue with Step 3.


## 3. Helm

This step extends the service catalog:

* Generates an umbrella Helm chart in `managed/`
* Creates a cluster-specific overlay in `customer/`

```bash
kubara generate --helm
```


There are several helm chart values.yaml files with dummy `change-me` values, that need to be overwritten.
Example:
```yaml
# ... previous content of yaml file
admin: change-me
# ... rest of yaml
```
These are located in the 
`/customer-service-catalog/helm/<cluster>/<chart>/values.yaml` 
directory.

The chart directoriees, where the values.yamls files need to be edited are: 

* argo-cd
* cert-manager
* external-dns
* external-secrets
* homer-dashboard
* ingress-nginx
* kube-prometheus-stack
* kyverno-policy-reporter
* kyverno
* loki
* longhorn
* metallb
* metrics-server
* oauth2-proxy


> ⚠️ **Don't forget to commit and push your changes to Git!**

---

## 4. Deploying Argo CD

### 4.1 Bootstrap the Control Plane

> ⚠️ This command requires access to a Kubernetes cluster and, by default, uses the environment's kubeconfig.
> To target a specific cluster, provide your own config with `--kubeconfig your-kubeconfig`

```bash
kubara  --bootstrap-argocd --with-es-crds --with-prometheus-crds
```

Your platform should now be fully operational.

---

## 5. Access the Argo CD Dashboard

> **Username:** `wizard`
> **Password:** From `.env` (`ARGOCD_WIZARD_ACCOUNT_PASSWORD`)

1. Start port-forwarding:

   ```bash
   kubectl port-forward svc/argocd-server -n argocd 8080:443
   ```

2. Open your browser at: [https://localhost:8080](https://localhost:8080)

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
./kubara --config-file config2.yaml init --env-file .env2
```

This will use the values from `.env2` to generate `config2.yaml`.

Render Terraform modules and Helm charts for the new ControlPlane:

```bash
./kubara generate --terraform --config-file config2.yaml
./kubara generate --helm --config-file config2.yaml
```

Finally, bootstrap your additional ControlPlane:

```bash
./kubara --bootstrap-argocd --with-es-crds --with-prometheus-crds --env-file .env2
```

## What's Next?

After bootstrapping your platform, you can:

* [Add Argo CD projects](../2_managing_your_platform/add_app_project.md)
* [Add Git repositories](../2_managing_your_platform/add_app_repository.md)
* [Add Argo CD applications](../2_managing_your_platform/add_application.md)
* [Add Argo CD appset](../2_managing_your_platform/add_appset.md)
* [Add SSO Configuration](../2_managing_your_platform/add_sso.md)
* [Add additional worker clusters](../2_managing_your_platform/add_worker_cluster.md)
