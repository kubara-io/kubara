{{- range (.Values.bootstrapValues).dockerPullSecrets }}
{{- include "templateLibrary.externalSecrets.dockerPullSecret-ces" . }}
{{- end }}
