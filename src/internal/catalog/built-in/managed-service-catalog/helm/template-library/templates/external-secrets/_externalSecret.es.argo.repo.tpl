{{- define "templateLibrary.externalSecrets.argocd.repository.remoteRef" }}
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
{{- end }}

{{- define "templateLibrary.externalSecrets.argocd.repository" }}
{{- $authMode := default "https" .authMode }}
{{- $credentialSecretKey := "pat" }}
{{- $credentialRemoteRef := .remoteRef }}
{{- if eq $authMode "ssh" }}
{{- $credentialSecretKey = "sshPrivateKey" }}
{{- $credentialRemoteRef = default .remoteRef .sshPrivateKeyRemoteRef }}
{{- else if eq $authMode "github-app" }}
{{- $credentialSecretKey = "githubAppPrivateKey" }}
{{- $credentialRemoteRef = default .remoteRef .githubAppPrivateKeyRemoteRef }}
{{- else if ne $authMode "https" }}
{{- fail (printf "unsupported Argo CD repository authMode %q for repository %q" $authMode .name) }}
{{- end }}
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
        type: {{ default "git" .repoType }}
        url: {{.url}}
        {{- if eq $authMode "https" }}
        username: {{ .username }}
        password: "{{ "{{" }} .pat }}"
        {{- else if eq $authMode "ssh" }}
        sshPrivateKey: "{{ "{{" }} .sshPrivateKey }}"
        {{- else if eq $authMode "github-app" }}
        githubAppID: {{ .githubAppID | quote }}
        githubAppInstallationID: {{ .githubAppInstallationID | quote }}
        githubAppPrivateKey: "{{ "{{" }} .githubAppPrivateKey }}"
        {{- if .githubAppEnterpriseBaseUrl }}
        githubAppEnterpriseBaseUrl: {{ .githubAppEnterpriseBaseUrl | quote }}
        {{- end }}
        {{- end }}
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
    - secretKey: {{ $credentialSecretKey }}
      remoteRef:
{{- include "templateLibrary.externalSecrets.argocd.repository.remoteRef" (dict "name" .name "remoteRef" $credentialRemoteRef) | nindent 8 }}
---
{{- end }}
