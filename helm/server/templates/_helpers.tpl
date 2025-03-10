##nginx
{{/*
Expand the name of the chart.
*/}}
{{- define "nginx.name" -}}
{{- default .Chart.Name .Values.nginx.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "nginx.fullname" -}}
{{- if .Values.nginx.fullnameOverride }}
{{- .Values.nginx.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nginx.nameOverride }}
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
{{- define "nginx.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nginx.labels" -}}
company: admin
{{ include "nginx.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "nginx.annotations" -}}
name: nginx
company: admin
maintainer: hongchun.you@admin.com
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nginx.selectorLabels" -}}
app: {{ .Values.nginx.fullnameOverride }} 
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "nginx.serviceAccountName" -}}
{{- if .Values.nginx.serviceAccount.create }}
{{- default (include "nginx.fullname" .) .Values.nginx.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.nginx.serviceAccount.name }}
{{- end }}
{{- end }}


####crash
{{- define "crash.name" -}}
{{- default .Chart.Name .Values.crash.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "crash.fullname" -}}
{{- if .Values.crash.fullnameOverride }}
{{- .Values.crash.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.crash.nameOverride }}
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
{{- define "crash.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}
:
{{/*
Common labels
*/}}
{{- define "crash.labels" -}}
company: admin
{{ include "crash.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "crash.annotations" -}}
name: crash
company: admin
maintainer: hongchun.you@admin.com
{{- end }}

{{/*
Selector labels
*/}}
{{- define "crash.selectorLabels" -}}
app: {{ .Values.crash.fullnameOverride }} 
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "crash.serviceAccountName" -}}
{{- if .Values.crash.serviceAccount.create }}
{{- default (include "crash.fullname" .) .Values.crash.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.crash.serviceAccount.name }}
{{- end }}
{{- end }}

## zookeeper
{{/*
Expand the name of the chart.
*/}}
{{- define "zookeeper.name" -}}
{{- default .Chart.Name .Values.zookeeper.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "zookeeper.fullname" -}}
{{- if .Values.zookeeper.fullnameOverride }}
{{- .Values.zookeeper.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.zookeeper.nameOverride }}
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
{{- define "zookeeper.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "zookeeper.labels" -}}
company: admin
{{ include "zookeeper.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "zookeeper.annotations" -}}
name: zookeeper
company: admin
maintainer: hongchun.you@admin.com
{{- end }}

{{/*
Selector labels
*/}}
{{- define "zookeeper.selectorLabels" -}}
app: {{ .Values.zookeeper.fullnameOverride }} 
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "zookeeper.serviceAccountName" -}}
{{- if .Values.zookeeper.serviceAccount.create }}
{{- default (include "zookeeper.fullname" .) .Values.zookeeper.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.zookeeper.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "zookeeper.podAntiAffinity" -}}
podAntiAffinity:
  requiredDuringSchedulingIgnoredDuringExecution:
    - labelSelector:
        matchExpressions:
          - key: "app"
            operator: In
            values: 
            - admin-zookeeper
            - elasticsearch-master
            - elasticsearch-data
            - admin-hadoop-nn
            - admin-hadoop-dn
            - admin-hbase-master
      topologyKey: "kubernetes.io/hostname"
{{- end }}

---

####redis
{{- define "redis.name" -}}
{{- default .Chart.Name .Values.redis.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "redis.fullname" -}}
{{- if .Values.redis.fullnameOverride }}
{{- .Values.redis.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.redis.nameOverride }}
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
{{- define "redis.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}
:
{{/*
Common labels
*/}}
{{- define "redis.labels" -}}
company: admin
{{ include "redis.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "redis.annotations" -}}
name: redis
company: admin
maintainer: hongchun.you@admin.com
{{- end }}

{{/*
Selector labels
*/}}
{{- define "redis.selectorLabels" -}}
app: {{ .Values.redis.fullnameOverride }} 
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "redis.serviceAccountName" -}}
{{- if .Values.redis.serviceAccount.create }}
{{- default (include "redis.fullname" .) .Values.redis.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.redis.serviceAccount.name }}
{{- end }}
{{- end }}


####kafka
####kafka
{{- define "kafka.name" -}}
{{- default .Chart.Name .Values.kafka.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kafka.fullname" -}}
{{- if .Values.kafka.fullnameOverride }}
{{- .Values.kafka.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.kafka.nameOverride }}
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
{{- define "kafka.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}
:
{{/*
Common labels
*/}}
{{- define "kafka.labels" -}}
company: admin
{{ include "kafka.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "kafka.annotations" -}}
name: kafka
company: admin
maintainer: hongchun.you@admin.com
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kafka.selectorLabels" -}}
app: {{ .Values.kafka.fullnameOverride }} 
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kafka.serviceAccountName" -}}
{{- if .Values.kafka.serviceAccount.create }}
{{- default (include "kafka.fullname" .) .Values.kafka.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.kafka.serviceAccount.name }}
{{- end }}
{{- end }}

{{- define "kafka.kraft.controllerQuorumVoters" -}}
{{- if .Values.kafka.kraft.controllerQuorumVoters -}}
    {{- include "common.tplvalues.render" (dict "value" .Values.kafka.kraft.controllerQuorumVoters "context" $) -}}
{{- else -}}
  {{- $controllerVoters := list -}}
  {{- $fullname := include "kafka.fullname" . -}}
  {{- $releaseNamespace :=.Release.Namespace  -}}
  {{- range $i := until (int .Values.kafka.replicaCount) -}}
  {{- $nodeId := add (int $i) (int $.Values.kafka.controller.minId) -}}
  {{- $nodeAddress := printf "%s-%d.%s-headless.%s.svc.%s:%d" $fullname (int $i) $fullname $releaseNamespace $.Values.kafka.clusterDomain (int $.Values.kafka.listeners.controller.containerPort) -}}
  {{- $controllerVoters = append $controllerVoters (printf "%d@%s" $nodeId $nodeAddress ) -}}
  {{- end -}}
  {{- join "," $controllerVoters -}}
{{- end -}}
{{- end -}}

##minio
{{/*
Expand the name of the chart.
*/}}
{{- define "minio.name" -}}
{{- default .Chart.Name .Values.minio.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "minio.fullname" -}}
{{- if .Values.minio.fullnameOverride }}
{{- .Values.minio.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.minio.nameOverride }}
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
{{- define "minio.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "minio.labels" -}}
company: admin
{{ include "minio.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "minio.annotations" -}}
name: minio
company: admin
maintainer: hongchun.you@admin.com
{{- end }}

{{/*
Selector labels
*/}}
{{- define "minio.selectorLabels" -}}
app: {{ .Values.minio.fullnameOverride }} 
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "minio.serviceAccountName" -}}
{{- if .Values.minio.serviceAccount.create }}
{{- default (include "minio.fullname" .) .Values.minio.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.minio.serviceAccount.name }}
{{- end }}
{{- end }}


##postgres
{{/*
Expand the name of the chart.
*/}}
{{- define "postgres.name" -}}
{{- default .Chart.Name .Values.postgres.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "postgres.fullname" -}}
{{- if .Values.postgres.fullnameOverride }}
{{- .Values.postgres.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.postgres.nameOverride }}
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
{{- define "postgres.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "postgres.labels" -}}
company: admin
{{ include "postgres.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "postgres.annotations" -}}
name: postgres
company: admin
maintainer: hongchun.you@admin.com
{{- end }}

{{/*
Selector labels
*/}}
{{- define "postgres.selectorLabels" -}}
app: {{ .Values.postgres.fullnameOverride }} 
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "postgres.serviceAccountName" -}}
{{- if .Values.postgres.serviceAccount.create }}
{{- default (include "postgres.fullname" .) .Values.postgres.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.postgres.serviceAccount.name }}
{{- end }}
{{- end }}
