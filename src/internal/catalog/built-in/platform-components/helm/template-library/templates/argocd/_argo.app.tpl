{{- define "templateLibrary.argocd.application" }}
{{- $ctx := index . 1 }}
{{- with index . 0 }}
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: "{{ .projectName }}-{{ .name }}"
  namespace: {{ .namespace }}
  annotations:
    {{- range $k,$v := .annotations }}
    {{- $k | quote | nindent 4 }}: {{ $v | quote}}
    {{- end }}
    valueFileChecksum: {{ (print .) | sha256sum}}
spec:
  project: {{ .projectName }}
  {{- if .info }}
  info:
  {{- include "templateLibrary.tpl-param" (dict "ctx" $ctx "param" .info) | indent 4 -}}
  {{- end }}
  {{- if .sources }}
  sources:
    {{- toYaml .sources | nindent 4 }}
  {{- else }}
  source:
    {{- if .directory }}
    directory:
      {{- toYaml .directory | nindent 6 }}
    {{- end }}
    repoURL: {{ .repoUrl }}
    targetRevision: {{ default "main" .targetRevision }}
    {{- if .chartName }}
    chart: {{ .chartName }}
    {{- else }}
    path: {{ default "." .repoPath }}
    {{- end }}
    {{- if or .appValues .extraValueFiles .appParameters }}
    helm:
      {{- if .extraValueFiles }}
      valueFiles:
        {{- range .extraValueFiles}}
        - {{ . | quote }}
        {{- end }}
      {{- end }}
      ignoreMissingValueFiles: true
      {{- if .appValues }}
      valuesObject:
        {{- tpl (toYaml .appValues) $ctx | nindent 8 }}
      {{- end }}
      {{- if .appParameters }}
      parameters:
      {{- include "templateLibrary.tpl-param" (dict "ctx" $ctx "param" .appParameters) | indent 8 }}
      {{- end }}
    {{- end }}
    {{- if .kustomize }}
    kustomize:
      {{- toYaml .kustomize | nindent 6 }}
    {{- end }}
  {{- end }}
  destination:
    {{- if (.destination).serverName }}
    name: {{ (.destination).serverName }}
    {{- else }}
    server: {{ default "https://kubernetes.default.svc" (.destination).server }}
    {{- end }}
    namespace: {{ default "argocd" (.destination).namespace }}
  syncPolicy:
    {{- if .syncPolicy }}
    {{- toYaml .syncPolicy | nindent 8 }}
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
    {{- end }}
  {{- if .ignoreDifferences }}
  ignoreDifferences:
    {{- with .ignoreDifferences }}
    {{- toYaml . | nindent 4 }}
    {{- end }}
  {{- end }}
  revisionHistoryLimit: {{ default 10 .revisionHistoryLimit }}
---
{{- end }}
{{- end }}
