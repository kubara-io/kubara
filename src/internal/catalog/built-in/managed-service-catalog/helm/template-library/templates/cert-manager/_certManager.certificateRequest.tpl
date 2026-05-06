{{- define "templateLibrary.certManager.CertificateRequest.tpl" -}}
apiVersion: cert-manager.io/v1
kind: CertificateRequest
metadata:
  name: {{ .Release.Name | printf "%s-%s" .Chart.Name }}
spec: {}
{{- end -}}
{{- define "templateLibrary.certManager.CertificateRequest" -}}
{{- include "templateLibrary.util.merge" (append . "templateLibrary.certManager.CertificateRequest.tpl") -}}
{{- end -}}
