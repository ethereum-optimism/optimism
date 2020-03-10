#!/usr/bin/env bats

load _helpers

@test "server/StatefulSet: disabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/StatefulSet: enable with --set" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'serverHA.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/StatefulSet: enable with global.enabled false" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'global.enabled=false' \
      --set 'serverHA.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/StatefulSet: disable with serverHA.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'serverHA.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/StatefulSet: disable with global.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'global.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/StatefulSet: image defaults to global.image" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'global.image=foo' \
      --set 'serverHA.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].image' | tee /dev/stderr)
  [ "${actual}" = "foo" ]
}

@test "server/StatefulSet: image can be overridden with serverHA.image" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'global.image=foo' \
      --set 'serverHA.enabled=true' \
      --set 'serverHA.image=bar' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].image' | tee /dev/stderr)
  [ "${actual}" = "bar" ]
}

##--------------------------------------------------------------------
## updateStrategy

@test "server/StatefulSet: no updateStrategy when not updating" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'serverHA.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.updateStrategy' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

@test "server/StatefulSet: updateStrategy during update" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'serverHA.enabled=true' \
      --set 'serverHA.updatePartition=2' \
      . | tee /dev/stderr |
      yq -r '.spec.updateStrategy.type' | tee /dev/stderr)
  [ "${actual}" = "RollingUpdate" ]

  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'serverHA.enabled=true' \
      --set 'serverHA.updatePartition=2' \
      . | tee /dev/stderr |
      yq -r '.spec.updateStrategy.rollingUpdate.partition' | tee /dev/stderr)
  [ "${actual}" = "2" ]
}

##--------------------------------------------------------------------
## extraVolumes

@test "server/StatefulSet: adds extra volume" {
  cd `chart_dir`

  # Test that it defines it
  local object=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'serverHA.enabled=true' \
      --set 'serverHA.extraVolumes[0].type=configMap' \
      --set 'serverHA.extraVolumes[0].name=foo' \
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
      -x templates/server-ha-statefulset.yaml  \
      --set 'serverHA.enabled=true' \
      --set 'serverHA.extraVolumes[0].type=configMap' \
      --set 'serverHA.extraVolumes[0].name=foo' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].volumeMounts[] | select(.name == "userconfig-foo")' | tee /dev/stderr)

  local actual=$(echo $object |
      yq -r '.readOnly' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(echo $object |
      yq -r '.mountPath' | tee /dev/stderr)
  [ "${actual}" = "/vault/userconfig/foo" ]

  # Doesn't load it
  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'serverHA.enabled=true' \
      --set 'serverHA.extraVolumes[0].type=configMap' \
      --set 'serverHA.extraVolumes[0].name=foo' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].command | map(select(test("userconfig"))) | length' | tee /dev/stderr)
  [ "${actual}" = "0" ]
}

@test "server/StatefulSet: adds extra secret volume" {
  cd `chart_dir`

  # Test that it defines it
  local object=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'serverHA.enabled=true' \
      --set 'serverHA.extraVolumes[0].type=secret' \
      --set 'serverHA.extraVolumes[0].name=foo' \
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
      -x templates/server-ha-statefulset.yaml  \
      --set 'serverHA.enabled=true' \
      --set 'serverHA.extraVolumes[0].type=configMap' \
      --set 'serverHA.extraVolumes[0].name=foo' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].volumeMounts[] | select(.name == "userconfig-foo")' | tee /dev/stderr)

  local actual=$(echo $object |
      yq -r '.readOnly' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(echo $object |
      yq -r '.mountPath' | tee /dev/stderr)
  [ "${actual}" = "/vault/userconfig/foo" ]

  # Doesn't load it
  local actual=$(helm template \
      -x templates/server-ha-statefulset.yaml  \
      --set 'serverHA.enabled=true' \
      --set 'serverHA.extraVolumes[0].type=configMap' \
      --set 'serverHA.extraVolumes[0].name=foo' \
      . | tee /dev/stderr |
      yq -r '.spec.template.spec.containers[0].command | map(select(test("userconfig"))) | length' | tee /dev/stderr)
  [ "${actual}" = "0" ]
}

# Extra volumes are not used for loading Vault configuration at this time
#@test "server/StatefulSet: adds loadable volume" {
#  cd `chart_dir`
#  local actual=$(helm template \
#      -x templates/server-ha-statefulset.yaml  \
#      --set 'serverHA.enabled=true' \
#      --set 'serverHA.extraVolumes[0].type=configMap' \
#      --set 'serverHA.extraVolumes[0].name=foo' \
#      --set 'serverHA.extraVolumes[0].load=true' \
#      . | tee /dev/stderr |
#      yq -r '.spec.template.spec.containers[0].command | map(select(test("/vault/userconfig/foo"))) | length' | tee /dev/stderr)
#  [ "${actual}" = "1" ]
#}
