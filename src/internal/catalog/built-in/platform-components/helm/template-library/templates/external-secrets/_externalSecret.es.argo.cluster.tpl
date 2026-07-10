{{- define "templateLibrary.argocd.cluster" }}
{{- $authMethod := default "clientCertificate" .authMethod -}}
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
        {{- if .namespaces }}
        namespaces: {{ .namespaces | join "," }}
        {{- end }}
        server: "{{ `{{ $k8sconfig := .config | fromYaml }}{{- $cluster := (index $k8sconfig.clusters 0) -}}{{ $cluster.cluster.server }}` }}"
        {{- if eq $authMethod "token" }}
        config: "{{ `{{ $k8sconfig := .config | fromYaml }}{{- $cluster := (index $k8sconfig.clusters 0) -}}{{- $user := (index $k8sconfig.users 0) -}}{{ printf \"{\\\"bearerToken\\\":%s,\\\"tlsClientConfig\\\":{\\\"caData\\\":%s,\\\"certData\\\":%s,\\\"insecure\\\":%s,\\\"keyData\\\":%s}}\" (index $user.user \"token\" | toJson) (index $cluster.cluster \"certificate-authority-data\" | toJson) (index $user.user \"client-certificate-data\" | toJson) \"false\" (index $user.user \"client-key-data\" | toJson) }}` }}"
        {{- else if eq $authMethod "clientCertificate" }}
        config: "{{ `{{ $k8sconfig := .config | fromYaml }}{{- $cluster := (index $k8sconfig.clusters 0) -}}{{- $user := (index $k8sconfig.users 0) -}}{{ printf \"{\\\"bearerToken\\\":\\\"\\\",\\\"tlsClientConfig\\\":{\\\"caData\\\":%s,\\\"certData\\\":%s,\\\"insecure\\\":%s,\\\"keyData\\\":%s}}\" (index $cluster.cluster \"certificate-authority-data\" | toJson) (index $user.user \"client-certificate-data\" | toJson) \"false\" (index $user.user \"client-key-data\" | toJson) }}` }}"
        {{- else }}
        {{- fail (printf "Unknown authMethod %q, must be either 'token' or 'clientCertificate'" $authMethod) }}
        {{- end }}
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
        nullBytePolicy: Fail
---
{{- end }}
