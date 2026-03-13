{{- define "templateLibrary.kyverno.ClusterPolicy.tpl" -}}
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: {{ .Release.Name | printf "%s-%s" .Chart.Name }}
spec: {}
{{- end -}}
{{- define "templateLibrary.kyverno.ClusterPolicy" -}}
{{- include "templateLibrary.util.merge" (append . "templateLibrary.kyverno.ClusterPolicy.tpl") -}}
{{- end -}}
