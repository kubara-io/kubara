{{- define "templateLibrary.externalSecrets.dockerPullSecret-ces" }}
apiVersion: external-secrets.io/v1
kind: ClusterExternalSecret
metadata:
  name: {{ .name }}-ces
spec:
  externalSecretName: {{ .name }}-es
  namespaceSelectors:
    {{- if .matchNamespaceLabels }}
    - matchLabels:
      {{- with .matchNamespaceLabels }}
        {{- toYaml . | nindent 8 }}
      {{- end }}
    {{- end }}
  refreshTime: 1m
  externalSecretSpec:
    refreshInterval: {{ default "5m" .refreshInterval }}
    secretStoreRef:
      kind: {{ .secretStoreRef.kind }}
      name: {{ .secretStoreRef.name }}
    target:
      name: {{ .name }}
      creationPolicy: Owner
      template:
        type: kubernetes.io/dockerconfigjson
        data:
          .dockerconfigjson: "{{ "{{" }} .dockerconfigjson }}"
    data:
      - secretKey: dockerconfigjson
        remoteRef:
          {{- if .remoteRef }}
          key: {{ .remoteRef.remoteKey }}
          {{- if .remoteRef.remoteKeyProperty }}
          property: {{ .remoteRef.remoteKeyProperty }}
          {{- end }}
          {{- else }}
          key: {{ .name }}
          {{- end }}
          conversionStrategy: Default
          decodingStrategy: None
          metadataPolicy: None
---
{{- end }}
