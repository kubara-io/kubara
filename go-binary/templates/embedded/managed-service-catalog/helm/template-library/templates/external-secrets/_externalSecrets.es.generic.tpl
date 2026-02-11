{{- define "templateLibrary.externalSecrets.es.generic" }}
{{- $ := . }}
{{- $globalStore := ($.Values.externalSecrets).secretStoreRef }}
{{- range $item := ($.Values.externalSecrets).secrets }}
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: {{ $item.name | default (printf "%s-es" $item.target ) }}
  namespace: {{ $.Release.Namespace }}
  labels:
    app.kubernetes.io/part-of: {{ $.Release.Name }}
spec:
  refreshInterval: {{ $item.refreshInterval | default "5m" }}
  {{- with $item.secretStoreRef | default $globalStore }}
  secretStoreRef:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  target:
    name: {{ $item.target }}
    creationPolicy: Owner
  {{- if $item.data }}
  data:
    {{- range $data_item := $item.data }}
    - secretKey: {{ $data_item.secretKey }}
      remoteRef:
        key: {{ $data_item.remoteKey }}
        {{- with $data_item.remoteKeyProperty }}
        property: {{ . }}
        {{- end }}
        {{- with $data_item.version }}
        version: {{ . }}
        {{- end }}
        conversionStrategy: Default
        decodingStrategy: None
        metadataPolicy: None
    {{- end }}
  {{- else if $item.dataFrom }}
  dataFrom:
    {{- range $dataFrom_item := $item.dataFrom }}
    - extract:
        key: {{ $dataFrom_item.remoteKey }}
        {{- with $dataFrom_item.remoteKeyProperty }}
        property: {{ . }}
        {{- end }}
        {{- with $dataFrom_item.version }}
        version: {{ . }}
        {{- end }}
        conversionStrategy: Default
        decodingStrategy: None
        metadataPolicy: None
    {{- end }}
  {{- end }}
---
{{- end }}
{{- end }}
