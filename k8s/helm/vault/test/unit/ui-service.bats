#!/usr/bin/env bats

load _helpers

@test "ui/Service: disabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/ui-service.yaml  \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "ui/Service: enable with global.enabled false" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/ui-service.yaml  \
      --set 'global.enabled=false' \
      --set 'server.enabled=true' \
      --set 'ui.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "ui/Service: disable with server.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/ui-service.yaml  \
      --set 'server.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "ui/Service: disable with ui.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/ui-service.yaml  \
      --set 'ui.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "ui/Service: disable with ui.service.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/ui-service.yaml  \
      --set 'ui.service.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "ui/Service: disable with global.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/ui-service.yaml  \
      --set 'global.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "ui/Service: disable with global.enabled and server.enabled on" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/ui-service.yaml  \
      --set 'global.enabled=false' \
      --set 'server.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "ui/Service: no type by default" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/ui-service.yaml  \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

@test "ui/Service: specified type" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/ui-service.yaml  \
      --set 'ui.service.type=LoadBalancer' \
      --set 'ui.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "LoadBalancer" ]
}
