{{- define "templateLibrary.argocd.applicationset" }}
{{- $globalCtx := index . 1 }}
{{- $localCtx := index . 0 }}
{{- range $localCtx.apps }}
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: {{ .name }}
  namespace: {{ $globalCtx.Release.Namespace }}
spec:
  generators:
    - clusters:
        selector:
          matchLabels:
            {{ .name }}: enabled
  syncPolicy:
    preserveResourcesOnDeletion: {{ default false .preserveResourcesOnDeletion }}
  template:
    metadata:
      name: "{{ `{{name}}` }}-{{ .name }}"
      annotations:
        argocd.argoproj.io/manifest-generate-paths: ".;.."
    spec:
      project: {{ default $localCtx.projectName .projectName }}
      sources:
        {{- if .sources }}
        {{- toYaml .sources | nindent 8 }}
        {{- else }}
        {{- with $localCtx.customerServices }}
        - repoURL: {{ .repoURL }}
          targetRevision: "{{ .targetRevision }}"
          ref: valuesRepo
        {{- end }}
        - repoURL: {{ $localCtx.managedServices.repoURL }}
          path: "{{ $localCtx.managedServices.path }}/{{.path}}"
          targetRevision: "{{ $localCtx.managedServices.targetRevision }}"
          helm:
            ignoreMissingValueFiles: true
            releaseName: {{ .name }}
            valueFiles:
              - "values.yaml"
              - "$valuesRepo/{{ $localCtx.customerServices.path }}/{{ `{{name}}` }}/{{ .path }}/values.yaml"
        {{- end }}
      destination:
        name: "{{ `{{name}}` }}"
        namespace: {{ default .name .namespace }}
      syncPolicy:
        {{- if .syncPolicy }}
        {{- toYaml .syncPolicy | nindent 8 -}}
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
      {{- if .ignoreApplicationDifferences }}
      ignoreApplicationDifferences:
      {{- toYaml .ignoreApplicationDifferences | nindent 8 }}
      {{- end }}
---
{{- end }}
{{- end }}
