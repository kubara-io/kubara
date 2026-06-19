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
   The most common way to use Velero is with an **S3-compatible** object storage target plus credentials from your secret backend. `backupStorage.create: true` lets kubara create the bucket and credentials where supported. On STACKIT this is the default when Velero is enabled (see below).
2. **Volume backup mode**  
   Choose the mode based on your provider's snapshot durability and your recovery goal:
   - `fs-backup`: Uses the Velero node-agent to back up volume contents to object storage. This is the default.
   - `csi-snapshot`: Uses CSI snapshots. Snapshot durability depends on the provider and CSI driver.
   - `csi-data-mover`: Creates CSI snapshots and moves their data to object storage through the Velero node-agent.
   More about File System Backups can be found in the official [Velero docs](https://velero.io/docs/v1.18/file-system-backup/).
3. **Recovery goal**  
   Be clear whether you are optimizing for namespace restore, cluster rebuild, disaster recovery, or migration.

!!! warning "File-system backup and CSI snapshots are mutually exclusive"
    File-system backup via Kopia and CSI volume snapshots are **mutually exclusive** — they cannot both be active for the same PVC at once. When `backupMode: fs-backup` is enabled, Velero uses file-system backup for all volumes and takes no CSI snapshots. Choose one approach before going to production; switching later may leave gaps in your backup history.

    References: [File-system backup](https://velero.io/docs/v1.18/file-system-backup/) · [CSI snapshots](https://velero.io/docs/v1.18/csi/)

!!! warning "CSI snapshots are not full backups — know your provider"
    A CSI snapshot is generally **not a full, standalone backup** and is **not necessarily independent of its source volume**. Snapshot durability depends on your cloud provider and CSI driver.

    Before relying on plain CSI snapshots, check your provider's block storage documentation. If snapshots are removed together with their source volume — or if you need true off-volume backups — use `backupMode: fs-backup` or `backupMode: csi-data-mover`, so the data is stored in object storage independently of volume state.

    For example, the [Open Telekom Cloud EVS documentation](https://docs.otc.t-systems.com/elastic-volume-service/dev-guide/deleting_an_evs_disk.html) states: "When you delete an EVS disk, all the disk data including the snapshots created for this disk will be deleted."

    References: [Velero CSI support](https://velero.io/docs/main/csi/) · [CSI Snapshot Data Movement](https://velero.io/docs/main/csi-snapshot-data-movement/)

### 3-2-1 Backups
On a more general note, your team should try to follow the 3-2-1 backup strategy, meaning:
3 Copies of your data
2 Different mediums, so not on the same disk as describe above
1 Offsite location, e.g different Region or even different Provider

This makes your setup more resilient against possible disasters.

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
          backupStorage:
            s3Url: https://object.storage.eu01.onstackit.cloud
```

For STACKIT, `backupStorage.create: true` makes kubara generate the dedicated Object Storage bucket (`bucket-velero-<name>-<stage>`) and credentials group. The bucket region defaults to `eu01` and can be changed through `backupStorage.region`. Set `backupStorage.s3Url` to the matching S3 endpoint for that region.

The generated Terraform writes the S3-compatible credentials into STACKIT Secrets Manager at path `<cluster-name>/<stage>/velero/velero_s3_credentials`, key `cloud`, in the form:

```toml
[default]
aws_access_key_id = <ACCESS_KEY_ID>
aws_secret_access_key = <SECRET_ACCESS_KEY>
```

So on STACKIT you need `status: enabled` and `backupStorage.s3Url` unless you intentionally change the backup mode, bucket region, or use an existing bucket.

To use an existing S3-compatible bucket instead, set `backupStorage.create: false` and provide the bucket connection details in the Velero config:

```yaml
config:
  backupStorage:
    create: false
    bucketName: my-velero-backups
    region: eu01
    s3Url: https://s3.example.com
```

With `backupStorage.create: false`, kubara does not generate the Terraform bucket or credentials. Provide the S3-compatible credentials yourself in your secret backend at path `<cluster-name>/<stage>/velero/velero_s3_credentials`, key `cloud`, using the same format shown above.

Then:

1. Run `kubara generate`
2. Review:
     - `customer-service-catalog/helm/<cluster-name>/velero/values.yaml`
     - `customer-service-catalog/helm/<cluster-name>/velero/additional-values.yaml`
3. Commit and push so Argo CD can deploy Velero
4. **Test a full backup and restore cycle immediately after setup**  
   Do not consider Velero operational until you have verified that backups actually work end-to-end. A misconfigured node-agent, CSI driver integration, or S3 endpoint can silently produce incomplete or empty backups with no visible error during backup creation. Restore to a test namespace and confirm that data is intact. Repeat this test after major changes (Velero upgrades, CSI driver updates, storage migrations).  
   References: [Backup reference](https://velero.io/docs/v1.18/backup-reference/) · [Restore reference](https://velero.io/docs/v1.18/restore-reference/) · [Disaster recovery](https://velero.io/docs/v1.18/disaster-case/)

With a healthy Velero setup, create a backup with:

```bash
velero backup create my-backup --wait
```
Follow progress of the backup process with:
```bash
velero backup describe my-backup
```
To restore from a backup:
```bash
velero restore create --from-backup my-backup --wait
```

These are the most simple commands Velero offers for backup. For production we advice you to create automated backups.
For more information on that have a look at the [official documentation](https://velero.io/docs/main/backup-reference/).
You can also create a cronjob via the `values.yaml` setting, with the help of the `additional-values.yaml` file. You can look [here](https://github.com/vmware-tanzu/helm-charts/blob/beb24e2081a90f19949630e001cc37c760281c40/charts/velero/values.yaml#L762), how this might look like.

Use `additional-values.yaml` for environment-specific overrides you want to keep next to the generated baseline.

### Custom `VolumeSnapshotClass` via `additional-values.yaml`

When you use `backupMode: csi-snapshot` or `backupMode: csi-data-mover`, Velero uses **CSI snapshots** instead of file-system backups.

kubara writes `volumeSnapshotClass.k8sProvider` into the generated Velero values based on `terraform.provider`.
If your environment is not covered by one of the built-in provider mappings, or if you need provider-specific fields that differ from the default, define your own `VolumeSnapshotClass` in `customer-service-catalog/helm/<cluster-name>/velero/additional-values.yaml`.
When `terraform.provider` is `none`, kubara does not select a built-in `VolumeSnapshotClass` provider.

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

### Crash Consistency
If you use file based backup, to backup a deployed database, please refer to the backup tools of choice for your database. File based backup is not crash-consistent. E.g use `pg_dump` or `mysqldump` instead.

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
