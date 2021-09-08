{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "kubernetes-secret-generator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubernetes-secret-generator.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kubernetes-secret-generator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "kubernetes-secret-generator.labels" -}}
helm.sh/chart: {{ include "kubernetes-secret-generator.chart" . }}
{{ include "kubernetes-secret-generator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "kubernetes-secret-generator.selectorLabels" -}}
name: {{ include "kubernetes-secret-generator.name" . }}
app.kubernetes.io/name: {{ include "kubernetes-secret-generator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "kubernetes-secret-generator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "kubernetes-secret-generator.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Define the namespace to watch
*/}}
{{- define "kubernetes-secret-generator.watchNamespace" -}}
{{- if and .Values.serviceAccount.create .Values.rbac.create (not .Values.rbac.clusterRole) -}}
    {{ default .Values.watchNamespace .Release.Namespace }}
{{- else -}}
    {{ .Values.watchNamespace }}
{{- end -}}
{{- end -}}

{{/*
Return the proper Docker Image Registry Secret Names
{{ include "kubernetes-secret-generator.images.pullSecrets" ( dict "images" (list .Values.image) "global" .Values.global) }}
*/}}
{{- define "kubernetes-secret-generator.images.pullSecrets" -}}
  {{- $pullSecrets := list }}

  {{- if .global }}
    {{- range .global.imagePullSecrets -}}
      {{- $pullSecrets = append $pullSecrets . -}}
    {{- end -}}
  {{- end -}}

  {{- range .images -}}
    {{- range .pullSecrets -}}
      {{- $pullSecrets = append $pullSecrets . -}}
    {{- end -}}
  {{- end -}}

  {{- if (not (empty $pullSecrets)) }}
imagePullSecrets:
    {{- range $pullSecrets }}
  - name: {{ . }}
    {{- end }}
  {{- end }}
{{- end -}}

{{ define "kubernetes-secret-generator.images.image" -}}
    {{ $registry := .root.Values.global.imageRegistry | default .Values.registry -}}
    {{ if $registry -}}
        {{ $registry }}/{{ .Values.repository }}:{{ .Values.tag | default .root.Chart.AppVersion }}
    {{- else -}}
        {{ .Values.repository }}:{{ .Values.tag | default .root.Chart.AppVersion }}
    {{- end }}
{{- end -}}
