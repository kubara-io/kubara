{{- define "templateLibrary.kyverno.Policy.tpl" -}}
apiVersion: kyverno.io/v1
kind: Policy
metadata:
  name: {{ .Release.Name | printf "%s-%s" .Chart.Name }}
spec: {}
{{- end -}}
{{- define "templateLibrary.kyverno.Policy" -}}
{{- include "templateLibrary.util.merge" (append . "templateLibrary.kyverno.Policy.tpl") -}}
{{- end -}}
