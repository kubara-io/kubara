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
platform-components/helm/my-new-servie-in-a-long-dir-name/

## **Add Templates to your new Chart (optional)**
To use pre-configured processes (e.g. get secrets from vault) you can leverage the Chart `template-library` inside 
`platform-components`.<br>
Take a look at other chart to find out how to use these templates.<br>
The general steps are:
1. Add template-library as a dependency to your Chart.yaml
2. add a template to your new Chart's "templates/" directory and include the templates you need from `template-libary` Chart
   (check other Chart templates to find out how)
3. update the new Chart's values and/or override values according to the templates you included

## **Add Override Values to your platform-configs**
Add Override Values to your `platform-configs`:<br>
platform-configs/my-cluster/helm/my-new-servie-in-a-long-dir-name/values.generated.yaml

Optional: Add one or more `values-*.yaml` files in the same chart folder for cluster-specific overrides.
For example, you can use `values-additional.yaml`, but the generated ApplicationSet will also pick up other files matching `values-*.yaml`.

## **Modify Argo CD overlays**
This is an example on how to add an AppSet to the hub cluster.
Add the following to your Argo CD overlay, typically `platform-configs/<hub-cluster-name>/helm/argo-cd/values-additional.yaml`.
```yaml
bootstrapValues:
  applicationSets:  # usually your existing hub cluster key (for example "<cluster>-<stage>")
    my-hub-dev:
      projectName: my-hub-dev
      platformComponents:
        repoURL: https://your-repo.example/managed.git
        path: platform-components/helm
        targetRevision: main
      platformConfigs:
        repoURL: https://your-repo.example/customer.git
        path: platform-configs
        targetRevision: main
      apps:
        my-new-service:
          name: my-new-service # This will determine the generated AppName
          path: my-new-servie-in-a-long-dir-name # This points to the directory you created for the chart inside platform-components
inClusterSecretLabels:
  my-new-service: enabled
    
```

This is meant to be added to the same directive where all pre-configured appSets are defined.
It will deploy the app to all Argo CD clusters that have the label `my-new-service: enabled` set 

## **Push your changes to git**
Do not forget to push your changes to the git repository that serves your Argo CD application.
If you let Argo CD manage itself, it will add the configured application to the cluster.

## **Run kubara bootstrap again (if Argo CD is not managing itself )**
If Argo CD is not managing itself (default, see `config.yaml` with `services.argocd.status: disabled`) altering Argo CD values will have no effect until you run the following again:
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
