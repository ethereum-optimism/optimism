#!/usr/bin/env bats

load _helpers

@test "server/DisruptionBudget: enabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-disruptionbudget.yaml  \
      --set 'server.ha.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/DisruptionBudget: disable with server.enabled" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-disruptionbudget.yaml  \
      --set 'globa.enabled=false' \
      --set 'server.ha.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/DisruptionBudget: disable with server.disruptionBudget.enabled" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-disruptionbudget.yaml  \
      --set 'server.ha.disruptionBudget.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/DisruptionBudget: disable with global.enabled" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-disruptionbudget.yaml  \
      --set 'global.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/DisruptionBudget: disable with injector.exernalVaultAddr" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-disruptionbudget.yaml  \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/DisruptionBudget: correct maxUnavailable with n=1" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-disruptionbudget.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'server.ha.replicas=1' \
      . | tee /dev/stderr |
      yq '.spec.maxUnavailable' | tee /dev/stderr)
  [ "${actual}" = "0" ]
}

@test "server/DisruptionBudget: correct maxUnavailable with n=3" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-disruptionbudget.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'server.ha.replicas=3' \
      . | tee /dev/stderr |
      yq '.spec.maxUnavailable' | tee /dev/stderr)
  [ "${actual}" = "1" ]
}

@test "server/DisruptionBudget: correct maxUnavailable with n=5" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-disruptionbudget.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'server.ha.replicas=5' \
      . | tee /dev/stderr |
      yq '.spec.maxUnavailable' | tee /dev/stderr)
  [ "${actual}" = "2" ]
}
