# Backup & Recovery

kubara is GitOps first:

- Use Git and `kubara generate` / `kubara bootstrap` to recreate desired platform state.
- Use your secret backend as the source of truth for credentials and sensitive values.
- Use Velero to recover runtime resources and persistent data that Git cannot recreate.

This page covers the kubara-specific path. For Velero installation details, command usage, and troubleshooting, use the official Velero documentation.

Your goal should be to keep as much desired state as possible in your GitOps repository and secret backend, and only rely on Velero for runtime data such as PVC contents.

## Velero And Kubara

kubara supports Velero as a built-in component for backup, restore, disaster recovery, and migration.

kubara covers:

- enabling Velero in cluster config
- generated overlay values and GitOps rollout
- a provider-driven `VolumeSnapshotClass` baseline
- documentation on how Velero fits into a GitOps recovery model

You still own:

- provider-specific backup storage configuration
- backup and restore runbooks
- schedules, retention, restore tests, and troubleshooting

## Before Enabling Velero

Decide these three things first:

1. Backup storage: Velero needs an S3-compatible object storage target plus credentials from your secret backend.
2. Volume backup mode: `fsBackupEnabled: true` uses file-system backup via the node agent. `fsBackupEnabled: false` uses CSI snapshots.
3. Recovery goal: Be clear whether you are optimizing for namespace restore, cluster rebuild, disaster recovery, or migration.

## Enable Velero

Example `config.yaml` for a custom S3-compatible target:

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

Add the S3 credentials to your secret backend as an AWS credentials file:

```toml
[default]
aws_access_key_id = <ACCESS_KEY_ID>
aws_secret_access_key = <SECRET_ACCESS_KEY>
```

For T Cloud Public CCE, copy the matching Velero block from `customer-service-catalog/terraform/<cluster-name>/openbao/secrets.tf-example` to `secrets.tf`; it writes `secret/velero/velero_s3_credentials` with property `cloud`.
The generated T Cloud Public values default to the Terraform-managed bucket `velero-<cluster-name>-<stage>`, region `eu-de`, and endpoint `https://obs.eu-de.otc.t-systems.com`. Override `s3BucketName`, `s3BucketRegion`, or `s3Url` only when you also changed the generated infrastructure values or target a different OBS region.

For the generic ClusterSecretStore path, store the same credentials file at remote key `velero_s3_credentials`, property `cloud`.
The generated `BackupStorageLocation` references the synchronized Kubernetes Secret `velero-credentials` with key `cloud`.

Then:

1. Run `kubara generate`.
2. Review `customer-service-catalog/helm/<cluster-name>/velero/values.yaml`.
3. Add environment-specific overrides to `customer-service-catalog/helm/<cluster-name>/velero/additional-values.yaml` when needed.
4. Commit and push so Argo CD can deploy Velero.
5. Create one backup and test one restore early.

## Custom VolumeSnapshotClass

When `fsBackupEnabled: false`, Velero uses CSI snapshots instead of file-system backups. kubara writes `volumeSnapshotClass.k8sProvider` into the generated Velero values based on `terraform.provider`.

The managed chart includes provider mappings for `stackit` and `t-cloud-public`. If your environment needs different snapshot settings, define a complete replacement in `additional-values.yaml`:

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

Keep `name: velero-csi` and the label `velero.io/csi-volumesnapshot-class: "true"` unless you intentionally want Velero to use a different class selection setup.

## Recovery Model

Velero should complement GitOps, not replace it.

| Source | Typical content |
| ------ | --------------- |
| Git + kubara + Argo CD | Cluster definitions, generated platform config, Helm values, ApplicationSets, managed manifests |
| Secret backend | External credentials, OAuth secrets, provider tokens, synced secret resources |
| Velero | Runtime Kubernetes resources and persistent volume data |

## Recommended Recovery Flow

1. Make sure you have access to Git, the secret backend, Velero object storage, and a working target cluster.
2. Bootstrap the cluster again and let Argo CD restore the declared platform state.
3. Use Velero to restore runtime resources and persistent data.
4. Verify that Argo CD, External Secrets, ingress, certificates, DNS, and stateful workloads are healthy.

For most teams, the best first test is restoring one non-critical namespace or workload.

## Official Velero Docs

- [Basic Install](https://velero.io/docs/v1.18/basic-install/)
- [Customize Install](https://velero.io/docs/v1.18/customize-installation/)
- [Backup Reference](https://velero.io/docs/v1.18/backup-reference/)
- [Restore Reference](https://velero.io/docs/v1.18/restore-reference/)
- [File System Backup](https://velero.io/docs/v1.18/file-system-backup/)
- [CSI support](https://velero.io/docs/v1.18/csi/)
- [Cluster migration](https://velero.io/docs/v1.18/migration-case/)
