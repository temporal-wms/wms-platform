{{/*
Expand the name of the chart.
*/}}
{{- define "wms-platform.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "wms-platform.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "wms-platform.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "wms-platform.labels" -}}
helm.sh/chart: {{ include "wms-platform.chart" . }}
{{ include "wms-platform.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
environment: {{ .Values.global.environment }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "wms-platform.selectorLabels" -}}
app.kubernetes.io/name: {{ include "wms-platform.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "wms-platform.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default "wms-platform" .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Service-specific labels
*/}}
{{- define "wms-platform.service.labels" -}}
app: {{ .serviceName }}
tier: backend
component: {{ .serviceConfig.type }}
{{- end }}

{{/*
Service-specific selector labels
*/}}
{{- define "wms-platform.service.selectorLabels" -}}
app: {{ .serviceName }}
{{- end }}

{{/*
Get image for a service
*/}}
{{- define "wms-platform.service.image" -}}
{{- $registry := .Values.global.imageRegistry }}
{{- $repository := .serviceConfig.image.repository }}
{{- $tag := .serviceConfig.image.tag }}
{{- printf "%s/%s:%s" $registry $repository $tag }}
{{- end }}
