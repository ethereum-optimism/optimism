#!/usr/bin/env bats

load _helpers

@test "injector/Service: service enabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/injector-service.yaml \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(helm template \
      --show-only templates/injector-service.yaml \
      --set 'injector.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "injector/Service: disable with global.enabled false" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/injector-service.yaml \
      --set 'global.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/injector-service.yaml \
      --set 'global.enabled=false' \
      --set 'injector.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}
