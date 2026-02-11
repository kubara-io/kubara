{{- define "templateLibrary.externalSecrets.vault-clusterSecretStore" }}
{{- range $idx, $data := .Values.clusterSecretStores }}
{{- $storeName := default (printf "store-%d" $idx ) .storeName }}
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: {{ $storeName }}
spec:
  provider:
    vault:
      server: {{ .server }}
      path: {{ .path }}
      version: "v2"
      auth:
        {{- with .auth.userPass }}
        userPass:
          path: {{ default "userpass" .path }}
          username: {{ .username }}
          {{- with .secretRef }}
          secretRef:
            namespace: {{ default $.Release.Namespace .namespace }}
            name: {{ .name }}
            key: {{ default "password" .key }}
          {{- end }}
        {{- end }}
---
{{- end }}
{{- end }}
