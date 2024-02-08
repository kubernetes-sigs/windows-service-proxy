{{/*
Expand the name of the chart.
*/}}
{{- define "kube-proxy.name" -}}
{{- default "kube-proxy-windows" .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kube-proxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kube-proxy.labels" -}}
helm.sh/chart: {{ include "kube-proxy.chart" . }}
{{ include "kube-proxy.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kube-proxy.selectorLabels" -}}
app.kubernetes.io/name: kube-proxy-windows
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
