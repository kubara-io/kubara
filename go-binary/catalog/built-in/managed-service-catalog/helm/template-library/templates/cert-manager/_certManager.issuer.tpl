{{- define "templateLibrary.certManager.Issuer.tpl" -}}
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: {{ .Release.Name | printf "%s-%s" .Chart.Name }}
spec: {}
{{- end -}}
{{- define "templateLibrary.certManager.Issuer" -}}
{{- include "templateLibrary.util.merge" (append . "templateLibrary.certManager.Issuer.tpl") -}}
{{- end -}}
