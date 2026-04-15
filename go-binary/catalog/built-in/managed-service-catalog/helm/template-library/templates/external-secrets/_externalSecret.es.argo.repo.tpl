{{- define "templateLibrary.externalSecrets.argocd.repository" }}
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: {{ .name }}-es
spec:
  refreshInterval: {{ default "5m" .refreshInterval }}
  secretStoreRef:
    kind: {{ .secretStoreRef.kind }}
    name: {{ .secretStoreRef.name }}
  target:
    name: {{.name}}-repo
    creationPolicy: Owner
    template:
      type: Opaque
      metadata:
        labels:
          argocd.argoproj.io/secret-type: repository
      data:
        name: {{.name}}
        type: {{ .repoType }}
        url: {{.url}}
        username: {{ .username }}
        password: "{{ "{{" }} .pat }}"
        {{- if .proxy }}
        proxy: {{.proxy}}
        {{- end }}
        {{- if .noProxy }}
        noProxy: {{.noProxy | quote }}
        {{- end }}
        {{- if .projectScope }}
        project: {{.projectScope}}
        {{- end }}
        {{- if .insecure }}
        insecure: {{ .insecure | toString }}
        {{- else }}
        insecure: "false"
        {{- end }}
  data:
    - secretKey: pat
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
