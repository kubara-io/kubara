# How to add an AppSet to Argo CD
Make sure you added the needed Project and Repositories. You should also think about setting appropriate RBAC on the
Project.

An Argo CD AppSet is a logical concept to create many Argo CD applications with just one manifest.<br>
This allows Users to spawn one service with different configurations on many namespaces and clusters.
For more information and possible configuration check:<br>

- https://github.com/argoproj/argo-cd/blob/master/docs/operator-manual/applicationset.yaml
- https://argo-cd.readthedocs.io/en/stable/user-guide/application-set/

## **Add Chart to your platform-components**
Add the Chart you want to add to your `platform-components`:<br>
platform-components/helm/my-new-service-in-a-long-dir-name/

## **Add Override Values to your platform-configs**
Add Override Values to your `platform-configs`:<br>
platform-configs/my-cluster/helm/my-new-service-in-a-long-dir-name/values-additional.yaml

Optional: Add one or more `values-*.yaml` files in the same chart folder for cluster-specific overrides.
For example, you can use `values-additional.yaml`, but the generated ApplicationSet will also pick up other files matching `values-*.yaml`.

## **Modify Argo CD overlays**
This is an example on how to add an AppSet to the hub cluster.
Add the following to your Argo CD overlay, typically `platform-configs/<hub-cluster-name>/helm/argo-cd/values-additional.yaml`.
```yaml
bootstrapValues:
  applicationSets:  # match your existing hub cluster key (for example "<cluster>-<stage>")
    my-hub-dev:
      apps:
        my-new-service:
          name: my-new-service # This will determine the generated AppName and label selector
          path: my-new-service-in-a-long-dir-name # Points to the directory you created for the chart inside platform-components/helm
inClusterSecretLabels:
  my-new-service: enabled
```

This is meant to be added to the same directive where all pre-configured appSets are defined.
It will deploy the app to all Argo CD clusters that have the label `my-new-service: enabled` set 

## **Push your changes to git**
Do not forget to push your changes to the git repository that serves your Argo CD application.
If you let Argo CD manage itself, it will add the configured application to the cluster.

## **Run kubara bootstrap again (if Argo CD is not managing itself )**
If Argo CD is not managing itself (see `config.yaml` with `argocd.selfManaged: disabled`), altering Argo CD values will have no effect until you run the following again:
```bash
kubara bootstrap <hub-cluster-name-from-config-yaml>
```

## **Add App from another repository**
If you want to add an application that is stored in another repository you can use the `sources:` directive. It supports all the fields Argo CD supports. Do not forget to add the repository to the allowed repositories in your project. Also check the docs: https://argo-cd.readthedocs.io/en/stable/user-guide/multiple_sources/#multiple-sources-for-an-application
```yaml
bootstrapValues:
  applicationSets:
    my-hub-dev:
      apps:
        akv2k8s:
          name: akv2k8s
          sources:
            - repoURL: "https://your-repo.de/with-overlay-values"
              targetRevision: "main"
              ref: valuesRepo
            - repoURL: https://charts.spvapi.no
              chart: akv2k8s
              targetRevision: "2.7.3"
              helm:
                valueFiles:
                  # Keep `{{name}}`: the AppSet controller injects the cluster name
                  - "$valuesRepo/platform-configs/{{name}}/helm/akv2k8s/values.generated.yaml"
```
