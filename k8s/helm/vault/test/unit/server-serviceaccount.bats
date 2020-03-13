#!/usr/bin/env bats

load _helpers

@test "server/ServiceAccount: specify annotations" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-serviceaccount.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.serviceAccount.annotations.foo=bar' \
      . | tee /dev/stderr |
      yq -r '.metadata.annotations["foo"]' | tee /dev/stderr)
  [ "${actual}" = "null" ]

  local actual=$(helm template \
      --show-only templates/server-serviceaccount.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'server.serviceAccount.annotations.foo=bar' \
      . | tee /dev/stderr |
      yq -r '.metadata.annotations["foo"]' | tee /dev/stderr)
  [ "${actual}" = "bar" ]

  local actual=$(helm template \
      --show-only templates/server-serviceaccount.yaml  \
      --set 'server.ha.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.metadata.annotations["foo"]' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

@test "server/ServiceAccount: disable with global.enabled false" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'global.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'global.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.standalone.enabled=true' \
      --set 'global.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/ServiceAccount: disable by injector.externalVaultAddr" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.standalone.enabled=true' \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}
