{{- define "templateLibrary.argocd.cluster" }}
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: {{ .name }}-es
  namespace: {{ default "argocd" .namespace }}
spec:
  refreshInterval: {{ default "5m" .refreshInterval }}
  secretStoreRef:
    kind: {{ .secretStoreRef.kind }}
    name: {{ .secretStoreRef.name }}
  target:
    name: {{ .name }}-cluster-secret
    creationPolicy: Owner
    template:
      metadata:
        {{- with .annotations }}
        annotations:
          {{- . | toYaml | nindent 10 }}
        {{- end }}
        labels:
          argocd.argoproj.io/secret-type: "cluster"
          {{- if .additionalLabels }}
          {{- .additionalLabels | toYaml | nindent 10 }}
          {{- end }}
      data:
        name: {{ .name }}
        {{- if .project }}
        project: {{ .project }}
        {{- end }}
        server: "{{ `{{ $k8sconfig := .config | fromYaml }}{{- $cluster := (index $k8sconfig.clusters 0) -}}{{ $cluster.cluster.server }}` }}"
        config: "{{ `{{ $k8sconfig := .config | fromYaml }}{{- $cluster := (index $k8sconfig.clusters 0) -}}{{- $user := (index $k8sconfig.users 0) -}}{{ printf \"{\\\"bearerToken\\\":\\\"\\\",\\\"tlsClientConfig\\\":{\\\"caData\\\":%s,\\\"certData\\\":%s,\\\"insecure\\\":%s,\\\"keyData\\\":%s}}\" (index $cluster.cluster \"certificate-authority-data\" | toJson) (index $user.user \"client-certificate-data\" | toJson) \"false\" (index $user.user \"client-key-data\" | toJson) }}` }}"
  data:
    - secretKey: config
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
