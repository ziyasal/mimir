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

{{- define "mimir.metaMonitoring.logs.client" -}}
{{- if .url -}}
url: {{ .url }}
{{- if .tenantId }}
tenantId: {{ .tenantId }}
{{- end }}
{{- if or .username .passwordSecretName }}
basicAuth:
{{- if .username }}
  username:
    name: {{ include "mimir.resourceName" (dict "ctx" $.ctx "component" "logs-instance-usernames") }}
    key: {{ .usernameKey }}
{{- end }}
{{- if .passwordSecretName }}
  password:
    name: {{ .passwordSecretName }}
    key: password
{{- end }}
{{- end }}
externalLabels:
  cluster: "{{ include "mimir.clusterName" $.ctx }}"
{{- end -}}
{{- end -}}