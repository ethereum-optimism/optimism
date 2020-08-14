{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to
this (by the DNS naming spec). If release name contains chart name it will
be used as a full name.
*/}}
{{- define "vault.fullname" -}}
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
{{- define "vault.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Expand the name of the chart.
*/}}
{{- define "vault.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Compute the maximum number of unavailable replicas for the PodDisruptionBudget.
This defaults to (n/2)-1 where n is the number of members of the server cluster.
Add a special case for replicas=1, where it should default to 0 as well.
*/}}
{{- define "vault.pdb.maxUnavailable" -}}
{{- if eq (int .Values.server.ha.replicas) 1 -}}
{{ 0 }}
{{- else if .Values.server.ha.disruptionBudget.maxUnavailable -}}
{{ .Values.server.ha.disruptionBudget.maxUnavailable -}}
{{- else -}}
{{- div (sub (div (mul (int .Values.server.ha.replicas) 10) 2) 1) 10 -}}
{{- end -}}
{{- end -}}

{{/*
Set the variable 'mode' to the server mode requested by the user to simplify
template logic.
*/}}
{{- define "vault.mode" -}}
  {{- if .Values.injector.externalVaultAddr -}}
    {{- $_ := set . "mode" "external" -}}
  {{- else if eq (.Values.server.dev.enabled | toString) "true" -}}
    {{- $_ := set . "mode" "dev" -}}
  {{- else if eq (.Values.server.ha.enabled | toString) "true" -}}
    {{- $_ := set . "mode" "ha" -}}
  {{- else if or (eq (.Values.server.standalone.enabled | toString) "true") (eq (.Values.server.standalone.enabled | toString) "-") -}}
    {{- $_ := set . "mode" "standalone" -}}
  {{- else -}}
    {{- $_ := set . "mode" "" -}}
  {{- end -}}
{{- end -}}

{{/*
Set's the replica count based on the different modes configured by user
*/}}
{{- define "vault.replicas" -}}
  {{ if eq .mode "standalone" }}
    {{- default 1 -}}
  {{ else if eq .mode "ha" }}
    {{- .Values.server.ha.replicas | default 3 -}}
  {{ else }}
    {{- default 1 -}}
  {{ end }}
{{- end -}}

{{/*
Set's up configmap mounts if this isn't a dev deployment and the user
defined a custom configuration.  Additionally iterates over any
extra volumes the user may have specified (such as a secret with TLS).
*/}}
{{- define "vault.volumes" -}}
  {{- if and (ne .mode "dev") (or (.Values.server.standalone.config) (.Values.server.ha.config)) }}
        - name: config
          configMap:
            name: {{ template "vault.fullname" . }}-config
  {{ end }}
  {{- range .Values.server.extraVolumes }}
        - name: userconfig-{{ .name }}
          {{ .type }}:
          {{- if (eq .type "configMap") }}
            name: {{ .name }}
          {{- else if (eq .type "secret") }}
            secretName: {{ .name }}
          {{- end }}
            defaultMode: {{ .defaultMode | default 420 }}
  {{- end }}
  {{- if .Values.server.volumes }}
    {{- toYaml .Values.server.volumes | nindent 8}}
  {{- end }}
{{- end -}}

{{/*
Set's a command to override the entrypoint defined in the image
so we can make the user experience nicer.  This works in with
"vault.args" to specify what commands /bin/sh should run.
*/}}
{{- define "vault.command" -}}
  {{ if or (eq .mode "standalone") (eq .mode "ha") }}
          - "/bin/sh"
          - "-ec"
  {{ end }}
{{- end -}}

{{/*
Set's the args for custom command to render the Vault configuration
file with IP addresses to make the out of box experience easier
for users looking to use this chart with Consul Helm.
*/}}
{{- define "vault.args" -}}
  {{ if or (eq .mode "standalone") (eq .mode "ha") }}
          - |
            sed -E "s/HOST_IP/${HOST_IP?}/g" /vault/config/extraconfig-from-values.hcl > /tmp/storageconfig.hcl;
            sed -Ei "s/POD_IP/${POD_IP?}/g" /tmp/storageconfig.hcl;
            /usr/local/bin/docker-entrypoint.sh vault server -config=/tmp/storageconfig.hcl {{ .Values.server.extraArgs }}
  {{ end }}
{{- end -}}

{{/*
Set's additional environment variables based on the mode.
*/}}
{{- define "vault.envs" -}}
  {{ if eq .mode "dev" }}
            - name: VAULT_DEV_ROOT_TOKEN_ID
              value: "root"
  {{ end }}
{{- end -}}

{{/*
Set's which additional volumes should be mounted to the container
based on the mode configured.
*/}}
{{- define "vault.mounts" -}}
  {{ if eq (.Values.server.auditStorage.enabled | toString) "true" }}
            - name: audit
              mountPath: /vault/audit
  {{ end }}
  {{ if or (eq .mode "standalone") (and (eq .mode "ha") (eq (.Values.server.ha.raft.enabled | toString) "true"))  }}
    {{ if eq (.Values.server.dataStorage.enabled | toString) "true" }}
            - name: data
              mountPath: /vault/data
    {{ end }}
  {{ end }}
  {{ if and (ne .mode "dev") (or (.Values.server.standalone.config)  (.Values.server.ha.config)) }}
            - name: config
              mountPath: /vault/config
  {{ end }}
  {{- range .Values.server.extraVolumes }}
            - name: userconfig-{{ .name }}
              readOnly: true
              mountPath: {{ .path | default "/vault/userconfig" }}/{{ .name }}
  {{- end }}
  {{- if .Values.server.volumeMounts }}
    {{- toYaml .Values.server.volumeMounts | nindent 12}}
  {{- end }}
{{- end -}}

{{/*
Set's up the volumeClaimTemplates when data or audit storage is required.  HA
might not use data storage since Consul is likely it's backend, however, audit
storage might be desired by the user.
*/}}
{{- define "vault.volumeclaims" -}}
  {{- if and (ne .mode "dev") (or .Values.server.dataStorage.enabled .Values.server.auditStorage.enabled) }}
  volumeClaimTemplates:
      {{- if and (eq (.Values.server.dataStorage.enabled | toString) "true") (or (eq .mode "standalone") (eq (.Values.server.ha.raft.enabled | toString ) "true" )) }}
    - metadata:
        name: data
      spec:
        accessModes:
          - {{ .Values.server.dataStorage.accessMode | default "ReadWriteOnce" }}
        resources:
          requests:
            storage: {{ .Values.server.dataStorage.size }}
          {{- if .Values.server.dataStorage.storageClass }}
        storageClassName: {{ .Values.server.dataStorage.storageClass }}
          {{- end }}
      {{ end }}
      {{- if eq (.Values.server.auditStorage.enabled | toString) "true" }}
    - metadata:
        name: audit
      spec:
        accessModes:
          - {{ .Values.server.auditStorage.accessMode | default "ReadWriteOnce" }}
        resources:
          requests:
            storage: {{ .Values.server.auditStorage.size }}
          {{- if .Values.server.auditStorage.storageClass }}
        storageClassName: {{ .Values.server.auditStorage.storageClass }}
          {{- end }}
      {{ end }}
  {{ end }}
{{- end -}}

{{/*
Set's the affinity for pod placement when running in standalone and HA modes.
*/}}
{{- define "vault.affinity" -}}
  {{- if and (ne .mode "dev") .Values.server.affinity }}
      affinity:
        {{ tpl .Values.server.affinity . | nindent 8 | trim }}
  {{ end }}
{{- end -}}

{{/*
Sets the injector affinity for pod placement
*/}}
{{- define "injector.affinity" -}}
  {{- if .Values.injector.affinity }}
      affinity:
        {{ tpl .Values.injector.affinity . | nindent 8 | trim }}
  {{ end }}
{{- end -}}

{{/*
Set's the toleration for pod placement when running in standalone and HA modes.
*/}}
{{- define "vault.tolerations" -}}
  {{- if and (ne .mode "dev") .Values.server.tolerations }}
      tolerations:
        {{ tpl .Values.server.tolerations . | nindent 8 | trim }}
  {{- end }}
{{- end -}}

{{/*
Sets the injector toleration for pod placement
*/}}
{{- define "injector.tolerations" -}}
  {{- if .Values.injector.tolerations }}
      tolerations:
        {{ tpl .Values.injector.tolerations . | nindent 8 | trim }}
  {{- end }}
{{- end -}}

{{/*
Set's the node selector for pod placement when running in standalone and HA modes.
*/}}
{{- define "vault.nodeselector" -}}
  {{- if and (ne .mode "dev") .Values.server.nodeSelector }}
      nodeSelector:
        {{ tpl .Values.server.nodeSelector . | indent 8 | trim }}
  {{- end }}
{{- end -}}

{{/*
Sets the injector node selector for pod placement
*/}}
{{- define "injector.nodeselector" -}}
  {{- if .Values.injector.nodeSelector }}
      nodeSelector:
        {{ tpl .Values.injector.nodeSelector . | indent 8 | trim }}
  {{- end }}
{{- end -}}

{{/*
Sets extra pod annotations
*/}}
{{- define "vault.annotations" -}}
  {{- if and (ne .mode "dev") .Values.server.annotations }}
      annotations:
        {{- $tp := typeOf .Values.server.annotations }}
        {{- if eq $tp "string" }}
          {{- tpl .Values.server.annotations . | nindent 8 }}
        {{- else }}
          {{- toYaml .Values.server.annotations | nindent 8 }}
        {{- end }}
  {{- end }}
{{- end -}}

{{/*
Sets extra ui service annotations
*/}}
{{- define "vault.ui.annotations" -}}
  {{- if .Values.ui.annotations }}
  annotations:
    {{- $tp := typeOf .Values.ui.annotations }}
    {{- if eq $tp "string" }}
      {{- tpl .Values.ui.annotations . | nindent 4 }}
    {{- else }}
      {{- toYaml .Values.ui.annotations | nindent 4 }}
    {{- end }}
  {{- end }}
{{- end -}}

{{/*
Sets extra service account annotations
*/}}
{{- define "vault.serviceAccount.annotations" -}}
  {{- if and (ne .mode "dev") .Values.server.serviceAccount.annotations }}
  annotations:
    {{- $tp := typeOf .Values.server.serviceAccount.annotations }}
    {{- if eq $tp "string" }}
      {{- tpl .Values.server.serviceAccount.annotations . | nindent 4 }}
    {{- else }}
      {{- toYaml .Values.server.serviceAccount.annotations | nindent 4 }}
    {{- end }}
  {{- end }}
{{- end -}}

{{/*
Sets extra ingress annotations
*/}}
{{- define "vault.ingress.annotations" -}}
  {{- if .Values.server.ingress.annotations }}
  annotations:
    {{- $tp := typeOf .Values.server.ingress.annotations }}
    {{- if eq $tp "string" }}
      {{- tpl .Values.server.ingress.annotations . | nindent 4 }}
    {{- else }}
      {{- toYaml .Values.server.ingress.annotations | nindent 4 }}
    {{- end }}
  {{- end }}
{{- end -}}

{{/*
Sets extra route annotations
*/}}
{{- define "vault.route.annotations" -}}
  {{- if .Values.server.route.annotations }}
  annotations:
    {{- $tp := typeOf .Values.server.route.annotations }}
    {{- if eq $tp "string" }}
      {{- tpl .Values.server.route.annotations . | nindent 4 }}
    {{- else }}
      {{- toYaml .Values.server.route.annotations | nindent 4 }}
    {{- end }}
  {{- end }}
{{- end -}}

{{/*
Sets extra vault server Service annotations
*/}}
{{- define "vault.service.annotations" -}}
  {{- if .Values.server.service.annotations }}
    {{- $tp := typeOf .Values.server.service.annotations }}
    {{- if eq $tp "string" }}
      {{- tpl .Values.server.service.annotations . | nindent 4 }}
    {{- else }}
      {{- toYaml .Values.server.service.annotations | nindent 4 }}
    {{- end }}
  {{- end }}
{{- end -}}

{{/*
Sets PodSecurityPolicy annotations
*/}}
{{- define "vault.psp.annotations" -}}
  {{- if .Values.global.psp.annotations }}
  annotations:
    {{- $tp := typeOf .Values.global.psp.annotations }}
    {{- if eq $tp "string" }}
      {{- tpl .Values.global.psp.annotations . | nindent 4 }}
    {{- else }}
      {{- toYaml .Values.global.psp.annotations | nindent 4 }}
    {{- end }}
  {{- end }}
{{- end -}}

{{/*
Set's the container resources if the user has set any.
*/}}
{{- define "vault.resources" -}}
  {{- if .Values.server.resources -}}
          resources:
{{ toYaml .Values.server.resources | indent 12}}
  {{ end }}
{{- end -}}

{{/*
Sets the container resources if the user has set any.
*/}}
{{- define "injector.resources" -}}
  {{- if .Values.injector.resources -}}
          resources:
{{ toYaml .Values.injector.resources | indent 12}}
  {{ end }}
{{- end -}}

{{/*
Inject extra environment vars in the format key:value, if populated
*/}}
{{- define "vault.extraEnvironmentVars" -}}
{{- if .extraEnvironmentVars -}}
{{- range $key, $value := .extraEnvironmentVars }}
- name: {{ printf "%s" $key | replace "." "_" | upper | quote }}
  value: {{ $value | quote }}
{{- end }}
{{- end -}}
{{- end -}}

{{/*
Inject extra environment populated by secrets, if populated
*/}}
{{- define "vault.extraSecretEnvironmentVars" -}}
{{- if .extraSecretEnvironmentVars -}}
{{- range .extraSecretEnvironmentVars }}
- name: {{ .envName }}
  valueFrom:
   secretKeyRef:
     name: {{ .secretName }}
     key: {{ .secretKey }}
{{- end -}}
{{- end -}}
{{- end -}}

{{/* Scheme for health check and local endpoint */}}
{{- define "vault.scheme" -}}
{{- if .Values.global.tlsDisable -}}
{{ "http" }}
{{- else -}}
{{ "https" }}
{{- end -}}
{{- end -}}
