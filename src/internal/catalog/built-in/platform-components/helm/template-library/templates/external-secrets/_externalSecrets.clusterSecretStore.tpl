{{- define "templateLibrary.externalSecrets.clusterSecretStores" }}
{{- range $name, $data := .Values.clusterSecretStores }}
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: {{ $name }}
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
