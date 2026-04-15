{{/* Expects a dictionary as input
key is the value (or Values) to be subsituted. Referenced by .value
value is the context which is used to find the substitution. Referenced by .context

Usage: {{ include "tpl-values" (dict "values" <value(s)> "ctx" $ ) }}
Example:
- in values.yaml -
overrides:
  url: "my-url.de"
url: "{{ .Values.overrides.url }}"

- in template -
valuesObject:
{{- include "tpl-values" (dict "values" ($.Files.Get $chartValues.valueFile) "ctx" $ ) | nindent 8 }}
*/}}
{{ define "templateLibrary.tpl-values" }}
    {{- if kindIs "string" .values }}
        {{- tpl .values .context  }}
    {{- else }}
        {{- tpl ( toYaml .values ) .context }}
    {{- end }}
{{- end }}

{{/* use like: {{- include "tpl-param" (dict "ctx" $ "array" $.Values.global.chart.parameters) | indent 8 }}
"ctx" is the context passed to the tpl function with
"array" must be an array of map[string] -> []interface {} with:
    name: string
    value: string --> value will be templated
*/}}
{{- define "templateLibrary.tpl-param" }}
{{- $ctx := .ctx }}
{{- range $_,$parameters := .param }}
- name: {{ get $parameters "name" | squote }}
  value: {{ tpl (get $parameters "value") $ctx | squote }}
{{- end }}
{{- end }}

{{/* merges b into a and prints merged content. Values of dict are templated.
*/}}
{{- define "templateLibrary.merge-tpl-dict" }}
{{- $ctx := default "" .ctx }}
{{- if or .a .b }}
{{- $global := dict }}
{{- $local := dict }}
{{- if .a -}}
{{- $global = mergeOverwrite (dict) .a }}
{{- end }}
{{- if .b }}
{{- $local = mergeOverwrite (dict) .b }}
{{- end }}
{{- $merge := mustMergeOverwrite $global $local }}
{{- range $k, $v := $merge }}
{{ $k }}: {{ tpl $v $ctx | quote }}
{{- end }}
{{- end }}
{{- end }}


{{/*
use like:
{{- $childChartContext := include "get-subchart-context" (dict "sub" "name-of-subchart" "ctx" $ ) | fromYaml }}

input: a dict with key -> val
    sub -> name of the subchart
    ctx -> the global context "$"

returns the subcharts context view; aka .Values, .Release, .Chart as as the given Subchart sees it
*/}}
{{- define "get-subschart-context" }}
{{- $subChart := .sub }}
{{- $ctx := .ctx }}
{{- with $ctx }}
{{- $childChartObjects := dict "Values" (index .Subcharts $subChart "Values") "Chart" (index .Subcharts $subChart "Chart") "Release" (index .Subcharts $subChart "Release") }}
{{- toYaml $childChartObjects }}
{{- end }}
{{- end }}
