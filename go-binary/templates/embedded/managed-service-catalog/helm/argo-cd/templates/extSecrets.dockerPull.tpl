{{- range $key, $secret := (.Values.bootstrapValues).dockerPullSecrets }}
{{- include "templateLibrary.externalSecrets.dockerPullSecret-ces" $secret }}
{{- end }}
