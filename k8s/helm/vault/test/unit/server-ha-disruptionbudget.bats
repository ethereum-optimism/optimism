#!/usr/bin/env bats

load _helpers

@test "server/DisruptionBudget: enabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-disruptionbudget.yaml  \
      --set 'serverHA.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/DisruptionBudget: enable with global.enabled false" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-disruptionbudget.yaml  \
      --set 'global.enabled=false' \
      --set 'serverHA.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/DisruptionBudget: disable with server.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-disruptionbudget.yaml  \
      --set 'serverHA.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/DisruptionBudget: disable with server.disruptionBudget.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-disruptionbudget.yaml  \
      --set 'server.disruptionBudget.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/DisruptionBudget: disable with global.enabled" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-disruptionbudget.yaml  \
      --set 'global.enabled=false' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/DisruptionBudget: correct maxUnavailable with n=3" {
  cd `chart_dir`
  local actual=$(helm template \
      -x templates/server-disruptionbudget.yaml  \
      --set 'serverHA.enabled=true' \
      --set 'serverHA.replicas=3' \
      . | tee /dev/stderr |
      yq '.spec.maxUnavailable' | tee /dev/stderr)
  [ "${actual}" = "0" ]
}
