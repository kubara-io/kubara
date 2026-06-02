{{- define "templateLibrary.externalSecrets.secretStore" }}
{{- range $idx, $data := .Values.namespacedSecretStores }}
{{- $storeName := default (printf "store-%d" $idx) (default $data.storeName $data.name) }}
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: {{ $storeName }}
  namespace: {{ $.Release.Namespace }}
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
