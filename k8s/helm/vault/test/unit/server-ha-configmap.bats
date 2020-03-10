#!/usr/bin/env bats

load _helpers

@test "server/ConfigMap: enabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-config-configmap.yaml  \
      --set 'serverHA.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/ConfigMap: enable with global.enabled false" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-config-configmap.yaml  \
      --set 'global.enabled=false' \
      --set 'serverHA.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/ConfigMap: disable with serverHA.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-config-configmap.yaml  \
      --set 'serverHA.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/ConfigMap: disable with global.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-config-configmap.yaml  \
      --set 'global.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/ConfigMap: extraConfig is set" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-ha-config-configmap.yaml  \
      --set 'serverHA.enabled=true' \
      --set 'serverHA.config="{\"hello\": \"world\"}"' \
      . | tee /dev/stderr |
      yq '.data["extraconfig-from-values.hcl"] | match("world") | length' | tee /dev/stderr)
  [ ! -z "${actual}" ]
}
