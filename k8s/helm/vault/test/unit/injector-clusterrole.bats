#!/usr/bin/env bats

load _helpers

@test "injector/ClusterRole: enabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/injector-clusterrole.yaml  \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "injector/ClusterRole: disable with global.enabled" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/injector-clusterrole.yaml  \
      --set 'global.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}
