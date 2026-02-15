{{- define "templateLibrary.externalSecrets.es.generic" }}
{{- $ := . }}
{{- $globalStore := ($.Values.externalSecrets).secretStoreRef }}
{{- $stores := ($.Values.externalSecrets).secretStores }}
{{- range $name, $item := ($.Values.externalSecrets).secrets }}
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: {{ $item.name | default (printf "%s-es" $name ) }}
  namespace: {{ $.Release.Namespace }}
  labels:
    app.kubernetes.io/part-of: {{ $.Release.Name }}
spec:
  refreshInterval: {{ $item.refreshInterval | default "5m" }}

  {{- $storeRef := $globalStore }}
  {{- if $item.secretStoreRef }}
    {{- if kindIs "string" $item.secretStoreRef }}
      {{- $storeRef = (get $stores $item.secretStoreRef) | default $globalStore }}
    {{- else }}
      {{- $storeRef = $item.secretStoreRef }}
    {{- end }}
  {{- end }}
  {{- with $storeRef }}
  secretStoreRef:
    {{- toYaml . | nindent 4 }}
  {{- end }}

  target:
    name: {{ $item.target | default $name }}
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
