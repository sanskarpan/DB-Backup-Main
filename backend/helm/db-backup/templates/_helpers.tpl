{{/*
Expand the name of the chart.
*/}}
{{- define "db-backup.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "db-backup.fullname" -}}
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
{{- define "db-backup.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "db-backup.labels" -}}
helm.sh/chart: {{ include "db-backup.chart" . }}
{{ include "db-backup.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "db-backup.selectorLabels" -}}
app.kubernetes.io/name: {{ include "db-backup.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: server
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "db-backup.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "db-backup.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the image name
*/}}
{{- define "db-backup.image" -}}
{{- $registry := .Values.global.imageRegistry | default .Values.image.registry -}}
{{- $repository := .Values.image.repository -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry $repository $tag }}
{{- else }}
{{- printf "%s:%s" $repository $tag }}
{{- end }}
{{- end }}

{{/*
Return the proper Docker Image Pull Secrets
*/}}
{{- define "db-backup.imagePullSecrets" -}}
{{- $pullSecrets := list }}
{{- if .Values.global.imagePullSecrets }}
{{- range .Values.global.imagePullSecrets }}
{{- $pullSecrets = append $pullSecrets (dict "name" .) }}
{{- end }}
{{- end }}
{{- if .Values.imagePullSecrets }}
{{- range .Values.imagePullSecrets }}
{{- $pullSecrets = append $pullSecrets (dict "name" .) }}
{{- end }}
{{- end }}
{{- if $pullSecrets }}
imagePullSecrets:
{{- range $pullSecrets }}
  - name: {{ .name }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Return the secret name
*/}}
{{- define "db-backup.secretName" -}}
{{- if .Values.secrets.create }}
{{- include "db-backup.fullname" . }}-secrets
{{- else }}
{{- .Values.secrets.externalSecretName }}
{{- end }}
{{- end }}

{{/*
Return the TLS secret name
*/}}
{{- define "db-backup.tlsSecretName" -}}
{{- if .Values.tls.create }}
{{- include "db-backup.fullname" . }}-tls
{{- else }}
{{- .Values.tls.secretName }}
{{- end }}
{{- end }}

{{/*
Return the configmap name
*/}}
{{- define "db-backup.configMapName" -}}
{{- include "db-backup.fullname" . }}-config
{{- end }}

{{/*
Return the PVC name
*/}}
{{- define "db-backup.pvcName" -}}
{{- include "db-backup.fullname" . }}-storage
{{- end }}
