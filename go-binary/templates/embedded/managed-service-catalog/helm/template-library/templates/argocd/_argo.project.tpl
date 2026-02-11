{{- define "templateLibrary.argocd.project" }}
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: {{ .name }}
  namespace: {{ default "argocd" .namespace  }}
  {{- if .finalizers }}
  finalizers:
  {{- range .finalizers }}
    - {{ . }}
  {{- end }}
  {{- end }}
  {{- if .annotations }}
  annotations:
    {{- range $k,$v := .annotations }}
    {{- $k | quote | nindent 4 }}: {{ $v | quote}}
    {{- end }}
  {{- end }}
spec:
  permitOnlyProjectScopedClusters: {{ default "false" .permitOnlyProjectScopedClusters }}
  {{- if .description }}
  description: |-
    {{- .description | nindent 4 }}
  {{- else }}
  description: {{ .name | title }} Project
  {{- end }}
  {{- with .sourceRepos }}
  sourceRepos:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  destinations:
  {{- if .destinations }}
    {{- toYaml .destinations | nindent 4 }}
  {{- else }}
    - server: "*"
      namespace: "*"
  {{- end }}
  sourceNamespaces:
  {{- if .sourceNamespaces }}
    {{- toYaml . | nindent 4 }}
  {{- else }}
    - "*"
  {{- end }}
  clusterResourceWhitelist:
  {{- if .clusterResourceWhitelist }}
    {{- toYaml .clusterResourceWhitelist | nindent 4 }}
  {{- else }}
    - group: "*"
      kind: "*"
  {{- end }}
  {{- with .clusterResourceBlacklist }}
  clusterResourceBlacklist:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .namespaceResourceBlacklist }}
  namespaceResourceBlacklist:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .namespaceResourceWhitelist }}
  namespaceResourceWhitelist:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .orphanedResources }}
  orphanedResources:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .roles }}
  roles:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .syncWindows }}
  syncWindows:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .signatureKeys }}
  signatureKeys:
    {{- toYaml . | nindent 4 }}
  {{- end }}
---
{{- end }}
