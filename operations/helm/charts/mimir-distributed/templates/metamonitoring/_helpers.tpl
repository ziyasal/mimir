{{- define "mimir.metaMonitoring.metrics.remoteWrite" -}}
{{- if .url -}}
url: {{ .url }}
{{- if or .username .passwordSecretName }}
basicAuth:
{{- if .username }}
  username:
    name: {{ include "mimir.resourceName" (dict "ctx" $.ctx "component" "metrics-instance-usernames") }}
    key: {{ .usernameKey }}
{{- end }}
{{- if .passwordSecretName }}
  password:
    name: {{ .passwordSecretName }}
    key: password
{{- end }}
{{- end }}
{{- end }}
{{- end -}}
