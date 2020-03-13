#!/usr/bin/env bats

load _helpers

@test "injector/ClusterRoleBinding: enabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/injector-clusterrolebinding.yaml  \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "injector/ClusterRoleBinding: disable with global.enabled" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/injector-clusterrolebinding.yaml  \
      --set 'global.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}
