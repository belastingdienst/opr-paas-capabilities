{{/*
paasargocd.singleLine renders a value as a quoted YAML string and fails the render when the value
contains control characters (newlines, tabs, ...).

Several values in this chart come from custom fields in a Paas, which are user input. Always render
such values through this helper (or at least `| quote`) so that a value can never break out of its
YAML field, for example by injecting a `---` document separator.

Usage: {{ include "paasargocd.singleLine" (list "git_url" .Values.git_url) }}
*/}}
{{- define "paasargocd.singleLine" -}}
{{- $name := index . 0 -}}
{{- $value := toString (index . 1) -}}
{{- if regexMatch "[[:cntrl:]]" $value -}}
{{- fail (printf "value %s must be a single line without control characters" $name) -}}
{{- end -}}
{{- $value | quote -}}
{{- end -}}
