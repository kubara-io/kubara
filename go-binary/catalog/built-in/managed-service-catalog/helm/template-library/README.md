# Ignore me
Just to test pipeline trigger#2

# Introduction
This library chart provides templates for creating k8s resources. This library-chart is meant to be added as a
dependency to your parent-chart. Once added you can use the library-templates within your parent-chart.  
This chart's purpose is to provide a centralized place for k8s-resource (templates) that get implemented by many
parent-charts. Additionally, it offers a layer of abstraction so that parent-charts can be kept DRY.

## Implementation
1. Add this library chart as a dependency to your parent-chart's _Chart.yaml_  
    ```yaml
    dependencies:
        - name: templateLibrary
          repository: <chart-repository>
          version: <version>
    ```
2. Run `helm dependency update`  

   You will find the contents of the library inside _charts/_ folder of your parent-chart.  
   Template helper-functions can be found inside _charts/templates/\_util.tpl_ and _charts/templates/\_functions.tpl_.

3. Implement a template from the library  
   
   In your parent-chart add a tpl or yaml file to your templates/ directory, e.g. templates/myExternalSecret.yaml and paste this
   into your file:
   ```gotemplate
   {{/* in parent-chart: templates/myExternalSecret.yaml */}}
   {{- include "templateLibrary.externalSecrets.ExternalSecret" (list . "test.ExternalSecret") -}}
   {{- define "test.ExternalSecret" -}}
   spec:
       myvalue: {{ .Values.test }}
   {{- end }}
   ```
   Note that the template names may differ depending on your custom implementations. The generic way to implement the 
   library-templates is:
   ```gotemplate
   {{- include "<name-of-template-in-your-library-chart>" (list . "<name-of-template-in-Parent-chart>") }}
   {{- define "<name-of-template-in-Parent>" -}}
   spec:
      someVariable: {{ .Values.Release.Namespace }}
      anotherVariable: "my string"
   {{- end }}
   ```

4. Run `helm template` on your parent-chart.  
   In addition to your parent-charts resources you will now see the created externalSecret resource in the output.

## Usage
You can include every named template from this chart inside your parent chart.

You can still introduce custom templates and other resources to your chart.
Make sure the names of the named templates from this library chart and your chart do not interfere.

### json schema