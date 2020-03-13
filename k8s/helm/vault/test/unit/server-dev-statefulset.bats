#!/usr/bin/env bats

load _helpers

@test "server/dev-StatefulSet: enable with server.dev.enabled true" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/dev-StatefulSet: disable with global.enabled" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'global.enabled=false' \
      --set 'server.dev.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/dev-StatefulSet: disable with injector.externalVaultAddr" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      --set 'server.dev.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/dev-StatefulSet: image defaults to server.image.repository:tag" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.image.repository=foo' \
      --set 'server.image.tag=1.2.3' \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].image' | tee /dev/stderr)
  [ "${actual}" = "foo:1.2.3" ]
}

@test "server/ha-StatefulSet: image tag defaults to latest" {
  cd `chart_dir`

  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.image.repository=foo' \
      --set 'server.image.tag=' \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].image' | tee /dev/stderr)
  [ "${actual}" = "foo:latest" ]
}

#--------------------------------------------------------------------
# replicas

@test "server/dev-StatefulSet: default replicas" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.replicas' | tee /dev/stderr)
  [ "${actual}" = "1" ]
}

@test "server/dev-StatefulSet: cant set replicas" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.dev.replicas=100' \
      . | tee /dev/stderr |
      yq -r '.spec.replicas' | tee /dev/stderr)
  [ "${actual}" = "1" ]
}

#--------------------------------------------------------------------
# updateStrategy

@test "server/dev-StatefulSet: updateStrategy" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.updateStrategy.type' | tee /dev/stderr)
  [ "${actual}" = "OnDelete" ]
}

#--------------------------------------------------------------------
# resources

@test "server/dev-StatefulSet: default resources" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].resources' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

@test "server/dev-StatefulSet: custom resources" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.resources.requests.memory=256Mi' \
      --set 'server.resources.requests.cpu=250m' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].resources.requests.memory' | tee /dev/stderr)
  [ "${actual}" = "256Mi" ]

  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.resources.limits.memory=256Mi' \
      --set 'server.resources.limits.cpu=250m' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].resources.limits.memory' | tee /dev/stderr)
  [ "${actual}" = "256Mi" ]

  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.resources.requests.cpu=250m' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].resources.requests.cpu' | tee /dev/stderr)
  [ "${actual}" = "250m" ]

  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.resources.limits.cpu=250m' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].resources.limits.cpu' | tee /dev/stderr)
  [ "${actual}" = "250m" ]
}

#--------------------------------------------------------------------
# extraVolumes

@test "server/dev-StatefulSet: adds extra volume" {
  cd `chart_dir`

  # Test that it defines it
  local object=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.extraVolumes[0].type=configMap' \
      --set 'server.extraVolumes[0].name=foo' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.volumes[] | select(.name == "userconfig-foo")' | tee /dev/stderr)

  local actual=$(echo $object |
      yq -r '.configMap.name' | tee /dev/stderr)
  [ "${actual}" = "foo" ]

  local actual=$(echo $object |
      yq -r '.configMap.secretName' | tee /dev/stderr)
  [ "${actual}" = "null" ]

  # Test that it mounts it
  local object=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.extraVolumes[0].type=configMap' \
      --set 'server.extraVolumes[0].name=foo' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].volumeMounts[] | select(.name == "userconfig-foo")' | tee /dev/stderr)

  local actual=$(echo $object |
      yq -r '.readOnly' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(echo $object |
      yq -r '.mountPath' | tee /dev/stderr)
  [ "${actual}" = "/vault/userconfig/foo" ]
}

@test "server/dev-StatefulSet: adds extra secret volume" {
  cd `chart_dir`

  # Test that it defines it
  local object=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.extraVolumes[0].type=secret' \
      --set 'server.extraVolumes[0].name=foo' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.volumes[] | select(.name == "userconfig-foo")' | tee /dev/stderr)

  local actual=$(echo $object |
      yq -r '.secret.name' | tee /dev/stderr)
  [ "${actual}" = "null" ]

  local actual=$(echo $object |
      yq -r '.secret.secretName' | tee /dev/stderr)
  [ "${actual}" = "foo" ]

  # Test that it mounts it
  local object=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.extraVolumes[0].type=configMap' \
      --set 'server.extraVolumes[0].name=foo' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].volumeMounts[] | select(.name == "userconfig-foo")' | tee /dev/stderr)

  local actual=$(echo $object |
      yq -r '.readOnly' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(echo $object |
      yq -r '.mountPath' | tee /dev/stderr)
  [ "${actual}" = "/vault/userconfig/foo" ]
}

@test "server/dev-StatefulSet: no storageClass on claim by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.volumeClaimTemplates[0].spec.storageClassName' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

#--------------------------------------------------------------------
# extraEnvironmentVars

@test "server/dev-StatefulSet: set extraEnvironmentVars" {
  cd `chart_dir`
  local object=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.extraEnvironmentVars.FOO=bar' \
      --set 'server.extraEnvironmentVars.FOOBAR=foobar' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].env' | tee /dev/stderr)

  local actual=$(echo $object |
     yq -r '.[7].name' | tee /dev/stderr)
  [ "${actual}" = "FOO" ]

  local actual=$(echo $object |
      yq -r '.[7].value' | tee /dev/stderr)
  [ "${actual}" = "bar" ]

  local actual=$(echo $object |
      yq -r '.[8].name' | tee /dev/stderr)
  [ "${actual}" = "FOOBAR" ]

  local actual=$(echo $object |
      yq -r '.[8].value' | tee /dev/stderr)
  [ "${actual}" = "foobar" ]
}

#--------------------------------------------------------------------
# extraSecretEnvironmentVars

@test "server/dev-StatefulSet: set extraSecretEnvironmentVars" {
  cd `chart_dir`
  local object=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.extraSecretEnvironmentVars[0].envName=ENV_FOO_0' \
      --set 'server.extraSecretEnvironmentVars[0].secretName=secret_name_0' \
      --set 'server.extraSecretEnvironmentVars[0].secretKey=secret_key_0' \
      --set 'server.extraSecretEnvironmentVars[1].envName=ENV_FOO_1' \
      --set 'server.extraSecretEnvironmentVars[1].secretName=secret_name_1' \
      --set 'server.extraSecretEnvironmentVars[1].secretKey=secret_key_1' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].env' | tee /dev/stderr)

  local actual=$(echo $object |
      yq -r '.[6].name' | tee /dev/stderr)
  [ "${actual}" = "ENV_FOO_0" ]
  local actual=$(echo $object |
      yq -r '.[6].valueFrom.secretKeyRef.name' | tee /dev/stderr)
  [ "${actual}" = "secret_name_0" ]
  local actual=$(echo $object |
      yq -r '.[6].valueFrom.secretKeyRef.key' | tee /dev/stderr)
  [ "${actual}" = "secret_key_0" ]

  local actual=$(echo $object |
      yq -r '.[7].name' | tee /dev/stderr)
  [ "${actual}" = "ENV_FOO_1" ]
  local actual=$(echo $object |
      yq -r '.[7].valueFrom.secretKeyRef.name' | tee /dev/stderr)
  [ "${actual}" = "secret_name_1" ]
  local actual=$(echo $object |
      yq -r '.[7].valueFrom.secretKeyRef.key' | tee /dev/stderr)
  [ "${actual}" = "secret_key_1" ]
}

#--------------------------------------------------------------------
# storage class

@test "server/dev-StatefulSet: can't set storageClass" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.dataStorage.enabled=true' \
      --set 'server.dataStorage.storageClass=foo' \
      . | tee /dev/stderr |
      yq -r '.spec.volumeClaimTemplates' | tee /dev/stderr)
  [ "${actual}" = "null" ]

  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.auditStorage.enabled=true' \
      --set 'server.auditStorage.storageClass=foo' \
      . | tee /dev/stderr |
      yq -r '.spec.volumeClaimTemplates' | tee /dev/stderr)
  [ "${actual}" = "null" ]

  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.auditStorage.enabled=true' \
      --set 'server.auditStorage.storageClass=foo' \
      --set 'server.auditStorage.enabled=true' \
      --set 'server.auditStorage.storageClass=foo' \
      . | tee /dev/stderr |
      yq -r '.spec.volumeClaimTemplates' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

#--------------------------------------------------------------------
# Security Contexts
@test "server/dev-StatefulSet: uid default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.securityContext.runAsUser' | tee /dev/stderr)
  [ "${actual}" = "100" ]
}

@test "server/dev-StatefulSet: uid configurable" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml \
      --set 'server.uid=2000' \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.securityContext.runAsUser' | tee /dev/stderr)
  [ "${actual}" = "2000" ]
}

@test "server/dev-StatefulSet: gid default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.securityContext.runAsGroup' | tee /dev/stderr)
  [ "${actual}" = "1000" ]
}

@test "server/dev-StatefulSet: gid configurable" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml \
      --set 'server.gid=2000' \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.securityContext.runAsGroup' | tee /dev/stderr)
  [ "${actual}" = "2000" ]
}

@test "server/dev-StatefulSet: fsgroup default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.securityContext.fsGroup' | tee /dev/stderr)
  [ "${actual}" = "1000" ]
}

@test "server/dev-StatefulSet: fsgroup configurable" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-statefulset.yaml \
      --set 'server.gid=2000' \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.securityContext.fsGroup' | tee /dev/stderr)
  [ "${actual}" = "2000" ]
}
