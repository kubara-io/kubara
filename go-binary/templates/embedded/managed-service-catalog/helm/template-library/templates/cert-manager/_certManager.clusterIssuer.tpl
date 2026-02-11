{{- define "templateLibrary.certManager.ClusterIssuer.tpl" -}}
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: {{ .Release.Name | printf "%s-%s" .Chart.Name }}
spec: {}
{{- end -}}
{{- define "templateLibrary.certManager.ClusterIssuer" -}}
{{- include "templateLibrary.util.merge" (append . "templateLibrary.certManager.ClusterIssuer.tpl") -}}
{{- end -}}
