{{- define "mimir.metaMonitoring.metrics.remoteWrite" -}}
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
{{- with .headers }}
headers:
  {{- toYaml . | nindent 2 }}
{{- end }}
{{- end -}}

{{- define "mimir.metaMonitoring.logs.client" -}}
url: {{ .url }}
{{- if .tenantId }}
tenantId: {{ .tenantId | quote }}
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