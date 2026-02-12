{{- define "templateLibrary.externalSecrets.clusterSecretStore" }}
{{- range $idx, $data := .Values.clusterSecretStores }}
{{- $storeName := default (printf "store-%d" $idx ) (default $data.storeName $data.name) }}
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: {{ $storeName }}
  {{- with $data.labels }}
  labels:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  provider:
    {{- toYaml $data.provider | nindent 4 }}
  {{- with $data.retrySettings }}
  retrySettings:
    {{- toYaml . | nindent 4 }}
  {{- end }}
---
{{- end }}
{{- end }}