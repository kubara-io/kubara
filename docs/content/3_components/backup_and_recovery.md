# Backup & Recovery

kubara is **GitOps first**:

- Use **Git** and `kubara generate` / `kubara bootstrap` to recreate desired platform state
- Use your **secret backend** as the source of truth for credentials and sensitive values
- Use **Velero** to recover runtime resources and persistent data that Git cannot recreate

This page covers the **kubara-specific path**. For Velero installation details, command usage, and troubleshooting, use the official Velero documentation.

Your goal should be to have as much of your desired state inside your GitOps repository and secret backend and only rely
on Velero for dynamic data that gets generated in runtime like the contents of PVCs of databases etc.

---

## Velero & kubara

kubara supports **Velero** as a built-in component for backup, restore, disaster recovery, and migration.

What kubara covers:

- Enabling Velero in cluster config
- Generated overlay files and GitOps rollout
- Documentation on how Velero fits into a recommended recovery model

What stays with you as a Platform Operator / Team:

- Provider-specific installation and configuration
- Backup and restore command tutorials and runbooks
- Scheduling, CSI, file-system backup internals, and troubleshooting

---

## Before enabling Velero

Decide these three things first:

1. **Backup storage**  
  The most common way to use velero is with a **S3-compatible** object storage target plus credentials from your secret backend.
2. **Volume backup mode**  
   `fsBackupEnabled: true` uses file-system backup via the node-agent.  
   `fsBackupEnabled: false` uses CSI snapshots instead.  
   More about File System Backups can be found in the official [Velero docs](https://velero.io/docs/v1.18/file-system-backup/).
3. **Recovery goal**  
   Be clear whether you are optimizing for namespace restore, cluster rebuild, disaster recovery, or migration.

!!! warning "File-system backup and CSI snapshots are mutually exclusive"
    `fsBackupEnabled: true` (file-system backup via Kopia) and CSI volume snapshots are **mutually exclusive** — they cannot both be active for the same PVC at once. When FSB is enabled globally, Velero uses file-system backup for all volumes and takes no CSI snapshots. Choose one approach before going to production; switching later may leave gaps in your backup history.

    References: [File-system backup](https://velero.io/docs/v1.18/file-system-backup/) · [CSI snapshots](https://velero.io/docs/v1.18/csi/)

!!! warning "Volume snapshots may not be true independent backups — know your provider"
    Volume snapshots are **not necessarily independent of the source volume**. On some cloud providers (e.g. Open Telekom Cloud / OTC), deleting a volume also deletes all its snapshots, so a CSI snapshot cannot protect you from accidental volume deletion or infrastructure-layer loss.

    Before relying on CSI snapshots, verify in your provider's block storage documentation whether snapshots survive the deletion of their source volume. If they do not, use file-system backup (`fsBackupEnabled: true`) to store data in S3, where it persists independently of volume state.

    References: verify with your provider's storage docs — e.g. [AWS EBS snapshots](https://docs.aws.amazon.com/ebs/latest/userguide/EBSSnapshots.html) are independent of their source volume; T Cloud Public EVS (Elastic Volume Service) snapshots are not.

---

## Enable Velero

Example `config.yaml`:

```yaml
clusters:
  - name: my-cluster
    stage: prod
    services:
      velero:
        status: enabled
        config:
          fsBackupEnabled: true
          s3BucketName: my-velero-backups
          s3BucketRegion: eu01
          s3Url: https://s3.example.com
```

Add your S3 credentials to your Secret Backend:

- Path: velero_s3_credentials
- Key: cloud 
- Contents:
  ```toml
  [default]
  aws_access_key_id = <ACCESS_KEY_ID>
  aws_secret_access_key = <SECRET_ACCESS_KEY>
  ```

Then:

1. Run `kubara generate`
2. Review:
     - `customer-service-catalog/helm/<cluster-name>/velero/values.yaml`
     - `customer-service-catalog/helm/<cluster-name>/velero/additional-values.yaml`
3. Commit and push so Argo CD can deploy Velero
4. **Test a full backup and restore cycle immediately after setup**  
   Do not consider Velero operational until you have verified that backups actually work end-to-end. A misconfigured node-agent, CSI driver integration, or S3 endpoint can silently produce incomplete or empty backups with no visible error during backup creation. Restore to a test namespace and confirm that data is intact. Repeat this test after major changes (Velero upgrades, CSI driver updates, storage migrations).  
   References: [Backup reference](https://velero.io/docs/v1.18/backup-reference/) · [Restore reference](https://velero.io/docs/v1.18/restore-reference/) · [Disaster recovery](https://velero.io/docs/v1.18/disaster-case/)

Use `additional-values.yaml` for environment-specific overrides you want to keep next to the generated baseline.

### Custom `VolumeSnapshotClass` via `additional-values.yaml`

When you use `fsBackupEnabled: false`, Velero uses **CSI snapshots** instead of file-system backups.

kubara writes `volumeSnapshotClass.k8sProvider` into the generated Velero values based on `terraform.provider`.
If your environment is not covered by one of the built-in provider mappings, or if you need provider-specific fields that differ from the default, define your own `VolumeSnapshotClass` in `customer-service-catalog/helm/<cluster-name>/velero/additional-values.yaml`.

`volumeSnapshotClass.customDefinition` takes precedence over the provider mapping, so this is the recommended way to supply a fully custom snapshot class.

Example:

```yaml
volumeSnapshotClass:
  customDefinition:
    apiVersion: snapshot.storage.k8s.io/v1
    kind: VolumeSnapshotClass
    metadata:
      name: velero-csi
      labels:
        velero.io/csi-volumesnapshot-class: "true"
    driver: ebs.csi.aws.com
    deletionPolicy: Retain
    parameters:
      tagSpecification_1: "Name=velero-snapshot"
```

Important notes:

- Keep `name: velero-csi` and the label `velero.io/csi-volumesnapshot-class: "true"` unless you intentionally want Velero to use a different class selection setup.
- Put this override into `additional-values.yaml`, not the generated `values.yaml`, because `values.yaml` is regenerated by kubara.
- If the built-in provider mapping already matches your environment, you usually do not need a custom definition.

---

## Recovery model

Velero should **complement** GitOps, not replace it.

| Source                 | Typical content                                                                                 |
| ---------------------- | ----------------------------------------------------------------------------------------------- |
| Git + kubara + Argo CD | Cluster definitions, generated platform config, Helm values, ApplicationSets, managed manifests |
| Secret backend         | External credentials, OAuth secrets, provider tokens, synced secret resources                   |
| Velero                 | Runtime Kubernetes resources and persistent volume data                                         |

---

## Recommended recovery flow

1. Required access to Git, the secret backend, the Velero object storage, and a working target cluster
2. Bootstrap the cluster again and let Argo CD restore the declared platform state
3. Use Velero to restore runtime resources and persistent data
4. Verify that Argo CD, External Secrets, ingress, certificates, DNS, and stateful workloads are healthy

For most teams, the best first test is restoring one non-critical namespace or workload.

---
## Misc.

### Other Storage Providers
Should you use a provider who does not support the S3 API, you can change the provider by replacing the plugin in `managed-service-catalog/helm/velero/values.yaml`. A list of available plugins can be found [here](https://velero.io/docs/v1.18/supported-providers/).

---

## Official Velero docs

- [Basic Install](https://velero.io/docs/v1.18/basic-install/)
- [Customize Install](https://velero.io/docs/v1.18/customize-installation/)
- [Backup Reference](https://velero.io/docs/v1.18/backup-reference/)
- [Restore Reference](https://velero.io/docs/v1.18/restore-reference/)
- [File System Backup](https://velero.io/docs/v1.18/file-system-backup/)
- [CSI support](https://velero.io/docs/v1.18/csi/)
- [Cluster migration](https://velero.io/docs/v1.18/migration-case/)
- [On-premises environments](https://velero.io/docs/v1.18/on-premises/)
