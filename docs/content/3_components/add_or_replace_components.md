# Add or replace components

With Kubara, you can add new components or replace existing ones. Of course, only the default components included in Kubara are supported and tested. 
No support can be provided for any components you add yourself.


## Add components

To do this, you can simply follow the instructions in the chapter: [add appset](../2_managing_your_platform/add_appset.md)

## Replace components

1. First, you need to disable the corresponding service in config.yaml. See chapter: [Bootstrap Your Platform](../1_getting_started/bootstrap_process.md)
2. Next, you need to re-template your Helm charts with Kubara (kubara generate --helm).
   Retemplating will also remove the corresponding entries in your values.yaml files in the customer-service-catalog folder. For example, if you disable Traefik, related ingress directives are removed from generated overlays.
3. Now you can add the new component (see above, adding components).
   Because the corresponding directives of the old component have now been removed from the values.yaml files, you must/can now set your own directives for your new component in the values.yaml files.
   For example, if you replace Traefik with another ingress controller, you need to set the new ingress directives yourself in your values files.
4. Then you need to commit and push all changes to your Git.
