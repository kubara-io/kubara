{{- define "templateLibrary.argocd.applicationset" }}
{{- $globalCtx := index . 1 }}
{{- $localCtx := index . 0 }}
{{- range $key, $app := $localCtx.apps }}
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: {{ $app.name }}
  namespace: {{ $globalCtx.Release.Namespace }}
spec:
  generators:
    - clusters:
        selector:
          matchLabels:
            {{ $app.name }}: enabled
  syncPolicy:
    preserveResourcesOnDeletion: {{ default false $app.preserveResourcesOnDeletion }}
  template:
    metadata:
      name: "{{ `{{name}}` }}-{{ $app.name }}"
      annotations:
        argocd.argoproj.io/manifest-generate-paths: ".;.."
    spec:
      project: {{ default $localCtx.projectName $app.projectName }}
      sources:
        {{- if $app.sources }}
        {{- toYaml $app.sources | nindent 8 }}
        {{- else }}
        {{- with $localCtx.customerServices }}
        - repoURL: {{ .repoURL }}
          targetRevision: "{{ .targetRevision }}"
          ref: valuesRepo
        {{- end }}
        - repoURL: {{ $localCtx.managedServices.repoURL }}
          path: "{{ $localCtx.managedServices.path }}/{{$app.path}}"
          targetRevision: "{{ $localCtx.managedServices.targetRevision }}"
          helm:
            ignoreMissingValueFiles: true
            releaseName: {{ $app.name }}
            valueFiles:
              - "values.yaml"
              - "$valuesRepo/{{ $localCtx.customerServices.path }}/{{ `{{name}}` }}/{{ $app.path }}/values.yaml"
              - "$valuesRepo/{{ $localCtx.customerServices.path }}/{{ `{{name}}` }}/{{ $app.path }}/additional-values.yaml"
        {{- end }}
      destination:
        name: "{{ `{{name}}` }}"
        namespace: {{ default $app.name $app.namespace }}
      syncPolicy:
        {{- if $app.syncPolicy }}
        {{- toYaml $app.syncPolicy | nindent 8 -}}
        {{- else }}
        automated:
          prune: true
          selfHeal: true
          allowEmpty: true
        syncOptions:
          - CreateNamespace=false
          - PruneLast=true
          - FailOnSharedResource=true
          - RespectIgnoreDifferences=true
          - ApplyOutOfSyncOnly=true
          - ServerSideApply=true
        {{- end }}
      {{- if $app.ignoreApplicationDifferences }}
      ignoreApplicationDifferences:
      {{- toYaml $app.ignoreApplicationDifferences | nindent 8 }}
      {{- end }}
---
{{- end }}
{{- end }}
