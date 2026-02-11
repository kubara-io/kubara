# Add AppSet to Argo CD
Make sure you added the needed Project and Repositories. You should also think about setting appropriate RBAC on the
Project.

An Argo CD AppSet is a logical concept to create many ArgoCD applications with just one manifest.<br>
This allows Users to spawn one service with different configurations on many namespaces and clusters.
For more information and possible configuration check:<br>
- https://github.com/argoproj/argo-cd/blob/master/docs/operator-manual/applicationset.yaml
- https://argo-cd.readthedocs.io/en/stable/user-guide/application-set/

## **Add Chart to your managed-service-catalog**
Add the Chart you want to add to your `managed-service-catalog`:<br>
managed-service-catalog/helm/my-new-servie-in-a-long-dir-name/

## **Add Templates to your new Chart (optional)**
To use pre-configured processes (e.g. get secrets from vault) you can leverage the Chart `template-library` inside 
`managed-service-catalog`.<br>
Take a look at other chart to find out how to use these templates.<br>
The general steps are:
1. Add template-library as a dependency to your Chart.yaml
2. add a template to your new Chart's "templates/" directory and include the templates you need from `template-libary` Chart
   (check other Chart templates to find out how)
3. update the new Chart's values and/or override values according to the templates you included

## **Add Override Values to your customer-service-catalog**
Add Override Values to your `customer-service-catalog`:<br>
customer-service-catalog/my-cluster/my-new-servie-in-a-long-dir-name/values.yaml

## **Modify argoCD overlays**
This is an example on how to add an AppSet to the controlplane.
Add the following to your `argo-cd/values.yaml`.
```yaml
bootstrapValues:
  applicationSets:
    - apps:
      - name: my-new-service # This will determine the generated AppName
        path: my-new-servie-in-a-long-dir-name # This points to the directory you created for the chart inside managed-service-catalog
inClusterSecretLabels:
  my-new-service: enabled
    
```

This is meant to be added to the same directive where all pre-configured appSets are defined.
It will deploy the app to all ArgoCD clusters that have the label `my-new-service: enabled` set 

## **Push your changes to git**
Do not forget to push your changes to the git repository that serves your argoCD application.
If you let argoCD manage itself, it will add the configured application to the cluster.

## **Run kubara bootstrap again (if ArgoCD is not managing itself )**
If ArgoCD is not managing itself (default, see config.yaml#services.argocd.enabled="disabled") altering argocd values will have no affect until you run
```bash
kubara --bootstrap-argocd
```
again

## **Add App from another repository**
If you want to add an application that is stored in another repository you can use the `sources:` directive. It supports all the fields argoCD supports. Do not forget to add the repository to the allowed repositories in your project. Also check the docs: https://argo-cd.readthedocs.io/en/stable/user-guide/multiple_sources/#multiple-sources-for-an-application
```yaml
bootstrapValues:
  applicationSets:
    - apps:
        - name: akv2k8s
          sources:
            - repoURL: https://charts.spvapi.no
              chart: akv2k8s
              targetRevision: "2.7.3"
              helm:
                valueFiles:
                # Kepp the `{{name}}`. The AppSet Controller will inject the clustername in the variable
                - "$valuesRepo/customer-service-catalog/helm/{{name}}/akv2k8s/values.yaml"
            - repoURL: "https://your-repo.de/with-overlay-values"
              targetRevision: "main"
              ref: valuesRepo
```