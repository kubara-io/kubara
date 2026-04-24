# STACKIT Provisioning Infrastructure (Edge Cloud)

Edge Cloud setups are usually highly individual.
Some teams run only STACKIT VMs, some run only bare metal, and others use mixed environments (VM + bare metal, even across different clouds).

kubara documents one possible integration path and examples, but your target setup may require different operational steps.

STEC (STACKIT Edge Cloud) can be operated through UI, CLI, or API.
All paths are valid:

- UI is often the fastest manual path for image and cluster creation.
- CLI/API with manifests is better for reproducibility and change history (for example in Git).

Choose the operating model that fits your team. kubara examples use manifest-based flows for traceability, but this is not mandatory.

## Optional Terraform modules for Edge

Terraform modules for Edge are optional and can be combined as needed:

* `edge_instance` is optional. `edge_instance.create` acts as module toggle:
  * `true`: create a new instance
  * `false` + `instance_id`: reuse existing instance
  * `false` + empty `instance_id`: skip module
* `edge_image` is optional. `edge_image.create` toggles image upload; upload runs only when `local_file_path` is set.
  In many setups, you can skip `edge_image` and reuse an existing image ID via `edge_hosts.image_id`.
* `edge_hosts` is optional. `edge_hosts.create` toggles host provisioning; host provisioning requires `edge_hosts.image_id` and at least one entry in `nodes`.
  The generated tfvars starts with a single-node bootstrap example (`controlplane` with public IP).
* `edge_hosts` does not auto-link to `edge_image`. Set `edge_hosts.image_id` explicitly:
  * use an existing image ID, or
  * upload via `edge_image` first, then copy `edge_uploaded_image_id` output into `edge_hosts.image_id`.

If your Edge platform is already provisioned externally, you can skip Terraform Edge modules completely and continue with kubara Helm/GitOps workflows only.

`EdgeImage` and `EdgeCluster` resources are managed via Kubernetes API (`kubectl`) and intentionally not part of the Terraform state in this setup.

Important distinction:

* `EdgeImage` and `image-factory` versions describe STEC boot artifacts and profiles (for example Talos version and extensions).
* `edge_hosts.image_id` must be a STACKIT project image ID (Compute/IaaS image), not an `image-factory` version string.

## Where to set `create = true/false`

Set the module toggles in the generated cluster tfvars file:

- `customer-service-catalog/terraform/<cluster-name>/infrastructure/env.auto.tfvars`

## Single-node bootstrap and network planning

The generated `edge_hosts.nodes` defaults to one control plane node as a bootstrap example.
This is intentionally simple for initial rollout and troubleshooting.

For production and multi-node topologies, design networking explicitly for your environment:

* where hosts run (VMs, bare metal, mixed, multi-cloud)
* ingress entry strategy (single public node, MetalLB VIP, external NLB, etc.)
* node-to-node traffic and return paths

There is no one-size-fits-all edge network design.

## Public IP and IP fields

In this example, Terraform allocates public IPs for hosts when `edge_hosts.nodes[*].assign_public_ip = true`.
You do not need to pre-enter a public IP for that path.

Read assigned host public IPs after apply via:

```bash
terraform output edge_host_metadata
```

Use the selected host public IP for DNS `A` records (for example your ingress hostname).

If you enable MetalLB, the generated MetalLB customer values use `clusters[].privateLoadBalancerIP` from your `config.yaml` as pool address (`/32`).
`publicLoadBalancerIP` is currently not wired into the generated MetalLB chart values.

## Step-by-step sequence

Before you start, go to:

```bash
cd customer-service-catalog/terraform/<cluster-name>/infrastructure
```

1. Run `terraform init` and `terraform plan` (or `tofu init` / `tofu plan`).
2. Configure `edge_instance` in `customer-service-catalog/terraform/<cluster-name>/infrastructure/env.auto.tfvars` (create or reuse).
3. Keep `edge_image.create = false` for the first apply.
4. Keep `edge_hosts.create = false` for the first apply (unless `edge_hosts.image_id` is already set and you want direct host provisioning).
5. Run first apply:

===  "Terraform"

    ```bash
    terraform apply
    ```

===  "Tofu"

    ```bash
    tofu apply
    ```

6. Check whether a suitable STACKIT project image already exists. If yes, set `edge_hosts.image_id`, keep `edge_image.create = false`, then continue with step 10.
7. If no suitable image exists, create `EdgeImage` in your STEC instance via `kubectl` and wait for `Ready`.
   If you plan to run Longhorn on this cluster, include `siderolabs/iscsi-tools` and `siderolabs/util-linux-tools` in the `EdgeImage` system extensions.
8. Read the generated artifact URL from `EdgeImage.status` and download it locally.
9. Set `edge_image.create = true` and set `edge_image.local_file_path` in `customer-service-catalog/terraform/<cluster-name>/infrastructure/env.auto.tfvars` to that local artifact path.
10. Run apply and read `edge_uploaded_image_id` from Terraform output, for example `terraform output edge_uploaded_image_id`.
11. Set `edge_hosts.image_id` to that ID (or keep your existing image ID), set `edge_hosts.create = true`, and configure `edge_hosts.nodes`.
12. Run apply again to create hosts.

===  "Terraform"

    ```bash
    terraform apply
    ```

===  "Tofu"

    ```bash
    tofu apply
    ```

13. Wait until the booted hosts are visible as `EdgeHost` in STEC.
14. Create `EdgeCluster` via `kubectl` and assign node roles (`controlplane` / `worker`).

How to read image information:

* List existing `EdgeImage` resources (STEC side):

```bash
kubectl get edgeimages.edge.stackit.cloud
kubectl get edgeimage <name> -o yaml
```

* List available `image-factory` Talos versions (STEC side):

```bash
INSTANCE_REGION="eu01"
curl https://image-factory.edge.$INSTANCE_REGION.stackit.cloud/versions
```

  These versions are not directly usable as `edge_hosts.image_id`.

* For Terraform host provisioning, you need a STACKIT project image ID for `edge_hosts.image_id`:
  * copy an existing ID from your STACKIT project image list (UI), or
  * use the Terraform upload module once and read:

```bash
terraform output edge_uploaded_image_id
```

If you use an existing image for Longhorn and know the related `EdgeImage`, verify that these extensions are present in that image configuration:

```bash
kubectl get edgeimage <name> -o yaml
# check spec.schematic -> customization.systemExtensions.officialExtensions
# expected: siderolabs/iscsi-tools and siderolabs/util-linux-tools
```

## Why this order

- `EdgeImage` is needed only when you do not already have a suitable STACKIT project image.
- Terraform `edge_image` upload needs a local artifact path.
- Terraform `edge_hosts` (if used) need an explicit `edge_hosts.image_id`.
- `EdgeCluster` should be created after hosts are registered as `EdgeHost`.
- You can source `edge_hosts.image_id` from an existing image or from `edge_uploaded_image_id` after a prior upload apply.
- In many setups, reusing an existing image ID is the default path; `edge_image` upload is only needed if you explicitly want to upload your own artifact.

## Talos and Kubernetes version compatibility

`EdgeImage` sets the Talos version used to build boot artifacts.
`EdgeCluster` sets Talos and Kubernetes versions for the cluster.

According to STACKIT docs:

* Talos version during cluster creation may differ from the boot image Talos version.
* If different, Talos automatically upgrades/downgrades during cluster creation.
* STEC validates that the chosen Kubernetes version is supported by the chosen Talos version.

Recommendation: keep `EdgeImage` Talos version and `EdgeCluster` Talos version aligned whenever possible to reduce moving parts during bootstrap.

Before applying manifests, verify which Talos image versions are currently available in your STEC region:

```bash
INSTANCE_REGION="eu01"
curl https://image-factory.edge.$INSTANCE_REGION.stackit.cloud/versions
```

Use one of the returned versions in both `EdgeImage.spec.talosVersion` and `EdgeCluster.spec.talos.version`.

## Longhorn extension requirement

If your kubara cluster will use Longhorn, your Talos image should include these system extensions:

- `siderolabs/iscsi-tools`
- `siderolabs/util-linux-tools`

Reason: Longhorn on Talos requires iSCSI tooling, and uses binaries from `util-linux-tools` (for example `fstrim`) for volume operations.
If your existing STACKIT project image was built from an `EdgeImage` profile that already includes these extensions, no extra rebuild is required.

## `EdgeImage` example

Use this to generate a Talos image in STEC:

```yaml
apiVersion: edge.stackit.cloud/v1alpha1
kind: EdgeImage
metadata:
  name: kubara-edge-image
  namespace: default
spec:
  schematic: |
    customization:
      extraKernelArgs: []
      systemExtensions:
        officialExtensions:
          - siderolabs/iscsi-tools
          - siderolabs/util-linux-tools
    overlay: {}
  talosVersion: v1.12.5-stackit.v1.7.1
```

If you are not using Longhorn, keep only the extensions you actually need.

## `EdgeCluster` example

Use this after your hosts are registered as `EdgeHost`:

```yaml
apiVersion: edge.stackit.cloud/v1alpha1
kind: EdgeCluster
metadata:
  name: kubara-cluster
  namespace: default
spec:
  nodes:
    - edgeHost: <edge-host-id-1>
      installDisk: /dev/vda
      role: controlplane
  talos:
    version: v1.12.5-stackit.v1.7.1
    kubernetes:
      version: v1.30.2
```

This minimal example reflects the single-node bootstrap path.
For production, add more control plane and worker nodes based on your network design.

## Validate API schema on your instance

`EdgeImage` and `EdgeCluster` schemas can evolve. Validate against your STEC instance before applying manifests:

```bash
kubectl api-resources --api-group=edge.stackit.cloud
kubectl explain edgeimages.edge.stackit.cloud.spec
kubectl explain edgeclusters.edge.stackit.cloud.spec
```

## Official references

* [Edge Cloud overview](https://docs.stackit.cloud/products/runtime/edge-cloud/)
* [Using the API](https://docs.stackit.cloud/products/runtime/edge-cloud/tutorials/using-the-api/)
* [Creating images](https://docs.stackit.cloud/de/products/runtime/edge-cloud/getting-started/creating-images/)
* [Using extensions](https://docs.stackit.cloud/products/runtime/edge-cloud/tutorials/using-extensions/)
* [Creating clusters](https://docs.stackit.cloud/products/runtime/edge-cloud/getting-started/creating-clusters/)
* [Authentication](https://docs.stackit.cloud/products/runtime/edge-cloud/getting-started/authentication/)
* [Longhorn Talos Linux support](https://longhorn.io/docs/1.11.0/advanced-resources/os-distro-specific/talos-linux-support/)

Now continue with the generic guide on the [Bootstrap Your Own Platform](../bootstrapping.md) page.
