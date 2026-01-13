{{/*
Expand the name of the chart.
*/}}
{{- define "wms-frontend.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "wms-frontend.fullname" -}}
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
{{- define "wms-frontend.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "wms-frontend.labels" -}}
helm.sh/chart: {{ include "wms-frontend.chart" . }}
{{ include "wms-frontend.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "wms-frontend.selectorLabels" -}}
app.kubernetes.io/name: {{ include "wms-frontend.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
App-specific labels
*/}}
{{- define "wms-frontend.app.labels" -}}
app.kubernetes.io/name: wms-{{ .appName }}
app.kubernetes.io/instance: {{ .appName }}
app.kubernetes.io/component: microfrontend
app.kubernetes.io/part-of: wms-frontend
{{- end }}

{{/*
App-specific selector labels
*/}}
{{- define "wms-frontend.app.selectorLabels" -}}
matchLabels:
  app.kubernetes.io/name: wms-{{ .appName }}
  app.kubernetes.io/instance: {{ .appName }}
{{- end }}

{{/*
Create image name
*/}}
{{- define "wms-frontend.app.image" -}}
{{- $registry := .Values.image.registry | default .Values.global.imageRegistry }}
{{- $repository := .Values.image.repository }}
{{- $tag := .Values.image.tag | default "latest" }}
{{- printf "%s/%s:%s" $registry $repository $tag }}
{{- end }}
