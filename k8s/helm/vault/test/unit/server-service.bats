#!/usr/bin/env bats

load _helpers

@test "server/Service: enabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-service.yaml  \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/Service: enable with global.enabled false" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-service.yaml  \
      --set 'global.enabled=false' \
      --set 'server.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/Service: disable with server.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-service.yaml  \
      --set 'server.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/Service: disable with global.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-service.yaml  \
      --set 'global.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

# This can be seen as testing just what we put into the YAML raw, but
# this is such an important part of making everything work we verify it here.
@test "server/Service: tolerates unready endpoints" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-service.yaml  \
      . | tee /dev/stderr |
      yq -r '.metadata.annotations["service.alpha.kubernetes.io/tolerate-unready-endpoints"]' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(helm template \
      -x templates/server-service.yaml  \
      . | tee /dev/stderr |
      yq -r '.spec.publishNotReadyAddresses' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}
