{{- define "templateLibrary.k8s.namespace" -}}
apiVersion: v1
kind: Namespace
metadata:
  name: {{ $.Release.Namespace }}
  {{- with .Values.namespace}}
  {{- with .labels}}
  labels:
     {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- with .annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
  {{- end }}
{{- end -}}
