{{- define "templateLibrary.certManager.Certificate.tpl" -}}
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: {{ .Release.Name | printf "%s-%s" .Chart.Name }}
spec: {}
{{- end -}}
{{- define "templateLibrary.certManager.Certificate" -}}
{{- include "templateLibrary.util.merge" (append . "templateLibrary.certManager.Certificate.tpl") -}}
{{- end -}}
