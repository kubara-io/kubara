{{- define "templateLibrary.externalSecrets.secretStores" }}
{{- range $name, $data := .Values.namespacedSecretStores }}
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: {{ $name }}
  namespace: {{ $.Release.Namespace }}
  {{- with $data.labels }}
  labels:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  provider:
    {{- toYaml $data.provider | nindent 4 }}
---
{{- end }}
{{- end }}
