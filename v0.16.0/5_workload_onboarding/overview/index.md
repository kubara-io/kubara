# Workload onboarding with Argo CD

This section is for workloads that should be added **on top of** your platform, not baked into the reusable platform catalog itself.

## When to use Argo CD instead of a catalog

Use the Argo CD path when the workload is:

- Specific to one cluster
- Specific to one team
- Specific to one product area
- An application that should live outside the reusable platform baseline

Hint:

- Use Argo CD workload onboarding when the service you are adding is a **cluster-specific or team-specific workload**.
- Use a catalog only when the service you are describing is part of the **reusable platform architecture**.  

## Which Argo CD guide should you use?

| Need | Guide |
| --- | --- |
| Limit what a team can deploy and from where | [How to add a Project](add_app_project.md) |
| Add a source repository | [How to add a Repository](add_app_repository.md) |
| Roll out one workload pattern to many clusters or namespaces | [How to add an AppSet](add_appset.md) |
| Add a single application or app-of-apps entry | [How to add an Application](add_application.md) |

## Rule of thumb

Ask this question:

> Am I changing the shared platform package, or am I adding a workload that should run on the platform?

If you are changing the shared platform package, go to the catalog docs.  
If you are adding a workload, stay in the Argo CD docs.

Related catalog pages:

- [Catalogs](../2_concepts/catalogs.md)
- [How to create a Catalog](../4_building_your_platform/create_catalog.md)
