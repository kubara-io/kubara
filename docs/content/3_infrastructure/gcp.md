# Kubara on GCP

If you want to run kubara on a hyperscaler like GCP, there are specific configurations to make.
Here we provide an example which can be conveniently adapted to other major clouds. 

In this example we use Provider native components like the google secretmanager for external-secrets, google dns with a delegated subdomain (from any domain registrar that allows setting NS records). For all other desired setups, please check the according Docs of the components. Also feel free to contribute! :)

We assume, you are starting in an empty project to test kubara.
Another assumption is that you made your self aware of the kubara deployment guide, so please read this guide and exactly follow the steps.

Please Note:
Velero on GCP wasn't tested, we recommend to follow the Guides of the Velero Project.


## Local Prerequisites

Before starting, the [gcloud CLI](https://cloud.google.com/sdk/docs/install) is required, along with an already authenticated account (`gcloud auth login`) with access to the target GCP project.

Obviously you need kubara too. See: [INSTALLATION GUIDE](../1_getting_started/installation.md1_getting_started/installation.md)

## Sneak preview:

We will create an configure a GKE Cluster, Google Secret Manager and use Google DNS with an existing domain, delegating a subdomain for External DNS and set up SSO.

We will check and adapt the following values files:

- gcp-test/platform-configs/gcp/helm/argo-cd/values-gcp.yaml. 
- gcp-test/platform-configs/gcp/helm/external-dns/values-gcp.yaml. 
- gcp-test/platform-configs/gcp/helm/external-secrets.yaml. 
- gcp-test/platform-configs/gcp/helm/kube-prometheus-stack/values-gcp.yaml. 
- gcp-test/platform-configs/gcp/helm/oauth2-proxy/values-gcp.yaml. 

We will then create our necessary secrets and deploy kubara on GKE.

Let's start!

## Part 1: Preparing kubara

```bash
# Install gke-gcloud-auth-plugin – required so kubectl can authenticate against the GKE cluster
gcloud components install gke-gcloud-auth-plugin

# Set the project
gcloud config set project $YOUR-PROJECT-NAME
```

Follow the kubara bootstrapping guide, generate your helm charts and stop before bootstrapping. Everything else we be handled in this Guide.
[Bootstrapping Guide](../1_getting_started/bootstrapping.md)


```bash
# pseudo-workflow, please check the official guide LINK

kubara init --prep # generate .env-file & set values
kubara init # generate config.yaml & set values
kubara generate # generate helm charts

# Stop after generating your Charts and proceed with ## Part 2
```

## TODO: Example config.yaml


## Part 2: GCP Infrastructure

### Enable Secret Manager

External DNS needs a secret store. As mentioned above, the platform-native Google Secret Manager is the natural choice. First, enable the corresponding API:

```bash
gcloud services enable secretmanager.googleapis.com
```

### Choose or Create a Network

Deploying Google Kubernetes Engine (GKE) requires a network (VPC). Existing networks can be checked as follows:

```bash
## check for existing networks/VPC
gcloud compute networks list
```

??? note "Optional: Example how to create a new network, router, subnet & Cloud NAT"
    If no suitable network exists, one can be created from scratch, including a router, a subnet (with secondary ranges for pods/services), and Cloud NAT. 
    Please be careful and adapt these commands to your needs before applying:

    ```bash
    ## OPTIONAL: create network & nat router
    gcloud compute networks create test-network --subnet-mode=auto

    ## router
    gcloud compute routers create test-router \
        --network=test-network \
        --region=europe-west3

    ## subnet
    gcloud compute networks subnets create test-subnet \
        --network=test-network \
        --region=europe-west3 \
        --range=10.0.0.0/24 \
        --secondary-range=pods=10.4.0.0/14,services=10.8.0.0/20

    ## cloud nat
    gcloud compute routers nats create test-nat \
        --router=test-router \
        --region=europe-west3 \
        --auto-allocate-nat-external-ips \
        --nat-all-subnet-ip-ranges
    ```

### Create the GKE Cluster

```bash
gcloud container clusters create test-cluster2 \
    --zone=europe-west3-a \                # Zonal cluster (single availability zone for testing)
    --network=test-network \                # VPC network the cluster is deployed into
    --subnetwork=test-subnet \              # Subnet within that network (incl. pod/service ranges)
    --enable-private-nodes \                # Nodes only get internal IPs, no public IP (more secure)
    --enable-ip-alias \                     # Enable VPC-native networking (required for private nodes & alias IP ranges)
    --master-ipv4-cidr=172.16.0.0/28 \      # Private IP range for the GKE control plane (must not overlap with anything else)
    --release-channel=regular \             # Update channel for automatic cluster upgrades (regular = balance between stability and new features)
    --machine-type=e2-standard-4 \          # VM type for worker nodes (4 vCPUs, 16 GB RAM)
    --num-nodes=1 \                         # Number of nodes at startup (per zone)
    --enable-autoscaling --min-nodes=0 --max-nodes=3 \  # Cluster autoscaler: scales automatically between 0 and 3 nodes
    --spot \                                # Uses cheap Spot VMs (can be reclaimed by the provider at any time – not suitable for critical workloads without tolerations)
    --disk-size=30 \                        # Boot disk size per node in GB
    --workload-pool=$(gcloud config get-value project).svc.id.goog \  # Enables Workload Identity: Kubernetes ServiceAccounts can securely impersonate GCP IAM service accounts, with no key files at all
    --async                                 # Command returns immediately without waiting for cluster creation to finish
```

### Restrict Control Plane / API Access to Your Own IP

`--enable-master-authorized-networks` adds a network-level layer that only allows explicitly permitted IP ranges to reach the API — following the principle of least privilege. For test purposes, your own public IP is enough; in a production environment, you would instead enter fixed CIDR ranges for bastion hosts, VPNs, CI/CD runners, etc..

```bash
## Restrict control plane / API access to your own IP:
gcloud container clusters update test-cluster2 \
    --zone=europe-west3-a \
    --enable-master-authorized-networks \                  # Restricts access to the control plane API to an allowlist of IP ranges
    --master-authorized-networks=$(curl -s ifconfig.me)/32 # Adds only your own public IP (as a /32, i.e. exactly one address) to that allowlist
```

### Wait for the Cluster & Check the Connection

```bash
## Wait until the cluster is ready (status: RUNNING)
gcloud container clusters list                                    # Lists all clusters in the project including their current status

## Generate credentials (kubeconfig) for the cluster
gcloud container clusters get-credentials test-cluster2 \
    --region=europe-west3-a                                       # Writes connection details for test-cluster2 into the local kubeconfig, so kubectl can access it

## Check the connection
kubectl get ns                                                     # Lists all namespaces in the cluster – fails if the connection/auth is broken
```

## Part 3: The Platform

### OAuth2 Configuration

How exactly these configurations are done depends on the chosen SSO provider (examples see the [SSO guide](../4_building_your_platform/sso/add_sso.md) for details) — regardless of the provider, however, at least the following SSO apps must be created for SSO:

1. Argo CD SSO
2. Grafana SSO
3. OAuth2 Proxy SSO

The resulting credentials are added manually later, in the [Create Secrets](#create-secrets) section. So save them for later.

### Note: Workload Identity
 We use workload identitys in these examples.
 Workload Identity binds a GCP service account directly and keylessly to a Kubernetes ServiceAccount. Pods then authenticate against GCP APIs transparently through that binding, with finely scoped IAM roles per use case — with no secret material to manage or rotate at all. (See: https://docs.cloud.google.com/iam/docs/workload-identities)

### external-DNS with Google DNS

```bash
# Create a service account for ExternalDNS
gcloud iam service-accounts create external-dns-sa \
    --display-name="SA for GKE ExternalDNS"

# Grant the service account write permissions on Cloud DNS
gcloud projects add-iam-policy-binding $(gcloud config get-value project) \
    --member="serviceAccount:external-dns-sa@$(gcloud config get-value project).iam.gserviceaccount.com" \
    --role="roles/dns.admin"

# Bind Workload Identity (note: requires the matching namespace – kubara's default is the chart name, so "external-dns" here)
gcloud iam service-accounts add-iam-policy-binding external-dns-sa@$(gcloud config get-value project).iam.gserviceaccount.com \
    --role="roles/iam.workloadIdentityUser" \
    --member="serviceAccount:$(gcloud config get-value project).svc.id.goog[external-dns/external-dns-sa]"

# Create the GCP managed zone
gcloud dns managed-zones create kubara-gcp-zone \
    --dns-name="subdomain.your-domain.com." \
    --description="Subdomain Zone for GKE kubara" \
    --visibility=public

# Print the zone's nameservers – these are then entered at the domain registrar as a delegated subdomain
gcloud dns managed-zones describe kubara-gcp-zone --format="value(nameServers)"
```

<!-- TODO: add reference to external-dns-values.yaml for the Helm chart -->
<!-- TODO: add example screenshot of a delegated subdomain at the registrar -->

Next, the `values.yaml` for external-dns needs to be adjusted.

See: `gcp-test/platform-configs/gcp/helm/external-dns/values-gcp.yaml`

```yaml
### set project id in service account annotation
external-dns:
  provider: google
  google:
    project: "<your-gcp-project-id>" # your project
  domainFilters:
    - "subdomain.your-domain.com" # your zone
  serviceAccount:
    create: true
    name: external-dns-sa
    annotations:
      iam.gke.io/gcp-service-account: "external-dns-sa@<your-gcp-project-id>.iam.gserviceaccount.com" # here!
```

### External Secrets

```bash
# 1. Create a Google service account
gcloud iam service-accounts create external-secrets-sa \
    --display-name="GKE Kubara External Secrets SA"

# 2. Grant the service account permission to read secrets (Secret Accessor)
gcloud projects add-iam-policy-binding $(gcloud config get-value project) \
    --member="serviceAccount:external-secrets-sa@$(gcloud config get-value project).iam.gserviceaccount.com" \
    --role="roles/secretmanager.secretAccessor"

# 3. Link the Kubernetes ServiceAccount via Workload Identity
gcloud iam service-accounts add-iam-policy-binding external-secrets-sa@$(gcloud config get-value project).iam.gserviceaccount.com \
    --role="roles/iam.workloadIdentityUser" \
    --member="serviceAccount:$(gcloud config get-value project).svc.id.goog[external-secrets/external-secrets-sa]"
```

#### SecretStore

See: `gcp-test/secretstore.yaml`

```yaml
# set your projectID and save!
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: gcp-store
  namespace: external-secrets
spec:
  provider:
    gcpsm:
      projectID: $YOUR-GCP-PROJECT-NAME
```

Caution!
Before `kubectl apply -f secretstore.yaml` can be run, the corresponding CRDs need to be installed first — however, this step is later handled automatically by the `kubara bootstrap` command, so don't try to apply now.

### Create Secrets

Now all secrets required by kubara are created. The following commands are just one proven example — secrets can just as well be created another way, as long as the name and content match.

#### Docker Config

```bash
gcloud secrets create gcp-dev-cluster-secrets-docker-config --replication-policy=automatic

# Decode the base64-encoded Docker pull secret and store it as JSON field "pull-secret" in Secret Manager
printf '%s' '$YOUR_PASSWORD_IN_BASE64' | base64 -d | jq -Rs '{"pull-secret":.}' | gcloud secrets versions add gcp-dev-cluster-secrets-docker-config --data-file=-
```

#### Grafana Admin Secret

```bash
gcloud secrets create gcp-dev-kube-prometheus-stack-grafana-credentials --replication-policy=automatic

printf '%s' '{"admin-user":"admin","admin-password":"YOUR_PASSWORD"}' | gcloud secrets versions add gcp-dev-kube-prometheus-stack-grafana-credentials --data-file=-
```

#### Grafana SSO Secret

```bash
gcloud secrets create gcp-dev-kube-prometheus-stack-grafana-oauth2-credentials --replication-policy=automatic

printf '%s' '{"client-id":"$YOUR_GRAFANA_CLIENT_ID","client-secret":"$YOUR_SECRET"}' | gcloud secrets versions add gcp-dev-kube-prometheus-stack-grafana-oauth2-credentials --data-file=-
```

#### OAuth2 Proxy Secret

```bash
gcloud secrets create gcp-dev-oauth2-proxy-oauth2-credentials --replication-policy=automatic

# Generate the cookie secret locally (see https://oauth2-proxy.github.io/oauth2-proxy/configuration/overview/)
dd if=/dev/urandom bs=32 count=1 2>/dev/null | base64 | tr -d -- '\n' | tr -- '+/' '-_' ; echo

printf '%s' '{"client-id":"$YOUR_OAUTH2_CLIENT_ID","client-secret":"$YOUR_SECRET","cookie-secret":"$YOUR_COOKIE_SECRET_CREATED_ABOVE"}' | gcloud secrets versions add gcp-dev-oauth2-proxy-oauth2-credentials --data-file=-
```

#### Argo CD SSO Secret

```bash
gcloud secrets create gcp-dev-argocd-argo-oauth2-credentials --replication-policy=automatic

printf '%s' '{"client-id":"$YOUR_ARGO_CLIENT_ID","client-secret":"$YOUR_SECRET"}' | gcloud secrets versions add gcp-dev-argocd-argo-oauth2-credentials --data-file=-
```

#### Velero
#### see: https://github.com/velero-io/velero-plugin-for-gcp#setup
Please refer to the official velero docs, we can only provide you these untested hints - feel free to contribute if you have some proposal to enhance the example.



### Part 4: Apply the SecretStore & Bootstrap Kubara

The `Secretstore` can be passed directly into the bootstrap process instead of applying via kubectl (just adjust the path/filename in `--with-es-css-file` if it differs):

```bash
kubara bootstrap gcp \
  --with-es-css-file secretstore.yaml \
  --with-es-crds --with-prometheus-crds
```

Now you should have a successfully deployed kubara installation.
Wait some minutes and check if all helm charts are synced and green.
Enjoy!