#!/usr/bin/env bats

load _helpers

@test "server/Service: service enabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.ha.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.standalone.enabled=true' \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/Service: disable with global.enabled false" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'global.enabled=false' \
      --set 'server.service.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'global.enabled=false' \
      --set 'server.service.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.standalone.enabled=true' \
      --set 'global.enabled=false' \
      --set 'server.service.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/Service: disable with server.service.enabled false" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'server.service.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'server.service.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.standalone.enabled=true' \
      --set 'server.service.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/Service: disable with global.enabled false server.service.enabled false" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'global.enabled=false' \
      --set 'server.service.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'global.enabled=false' \
      --set 'server.service.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.standalone.enabled=true' \
      --set 'global.enabled=false' \
      --set 'server.service.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/Service: disable with injector.externalVaultAddr" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      --set 'server.service.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      --set 'server.service.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/server-service.yaml  \
      --set 'server.standalone.enabled=true' \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      --set 'server.service.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

# This can be seen as testing just what we put into the YAML raw, but
# this is such an important part of making everything work we verify it here.
@test "server/Service: tolerates unready endpoints" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.metadata.annotations["service.alpha.kubernetes.io/tolerate-unready-endpoints"]' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.ha.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.metadata.annotations["service.alpha.kubernetes.io/tolerate-unready-endpoints"]' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.standalone.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.metadata.annotations["service.alpha.kubernetes.io/tolerate-unready-endpoints"]' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/Service: generic annotations" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.service.annotations.vaultIsAwesome=true' \
      . | tee /dev/stderr |
      yq -r '.metadata.annotations["vaultIsAwesome"]' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/Service: publish not ready" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.publishNotReadyAddresses' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.ha.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.publishNotReadyAddresses' | tee /dev/stderr)
  [ "${actual}" = "true" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.standalone.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.publishNotReadyAddresses' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "server/Service: type empty by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "null" ]

    local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.ha.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "null" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

@test "server/Service: type can set" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.dev.enabled=true' \
      --set 'server.service.type=NodePort' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "NodePort" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.ha.enabled=true' \
      --set 'server.service.type=NodePort' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "NodePort" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.service.type=NodePort' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "NodePort" ]
}

@test "server/Service: clusterIP empty by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.dev.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.clusterIP' | tee /dev/stderr)
  [ "${actual}" = "null" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.ha.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.clusterIP' | tee /dev/stderr)
  [ "${actual}" = "null" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      . | tee /dev/stderr |
      yq -r '.spec.clusterIP' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

@test "server/Service: clusterIP can set" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.dev.enabled=true' \
      --set 'server.service.clusterIP=None' \
      . | tee /dev/stderr |
      yq -r '.spec.clusterIP' | tee /dev/stderr)
  [ "${actual}" = "None" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.ha.enabled=true' \
      --set 'server.service.clusterIP=None' \
      . | tee /dev/stderr |
      yq -r '.spec.clusterIP' | tee /dev/stderr)
  [ "${actual}" = "None" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.service.clusterIP=None' \
      . | tee /dev/stderr |
      yq -r '.spec.clusterIP' | tee /dev/stderr)
  [ "${actual}" = "None" ]
}

@test "server/Service: port and targetPort will be 8200 by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      . | tee /dev/stderr |
      yq -r '.spec.ports[0].port' | tee /dev/stderr)
  [ "${actual}" = "8200" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      . | tee /dev/stderr |
      yq -r '.spec.ports[0].targetPort' | tee /dev/stderr)
  [ "${actual}" = "8200" ]
}

@test "server/Service: port and targetPort can be set" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.service.port=8000' \
      . | tee /dev/stderr |
      yq -r '.spec.ports[0].port' | tee /dev/stderr)
  [ "${actual}" = "8000" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.service.targetPort=80' \
      . | tee /dev/stderr |
      yq -r '.spec.ports[0].targetPort' | tee /dev/stderr)
  [ "${actual}" = "80" ]
}

@test "server/Service: nodeport can set" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.dev.enabled=true' \
      --set 'server.service.type=NodePort' \
      --set 'server.service.nodePort=30008' \
      . | tee /dev/stderr |
      yq -r '.spec.ports[0].nodePort' | tee /dev/stderr)
  [ "${actual}" = "30008" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.ha.enabled=true' \
      --set 'server.service.type=NodePort' \
      --set 'server.service.nodePort=30009' \
      . | tee /dev/stderr |
      yq -r '.spec.ports[0].nodePort' | tee /dev/stderr)
  [ "${actual}" = "30009" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.service.type=NodePort' \
      --set 'server.service.nodePort=30010' \
      . | tee /dev/stderr |
      yq -r '.spec.ports[0].nodePort' | tee /dev/stderr)
  [ "${actual}" = "30010" ]
}

@test "server/Service: nodeport can't set when type isn't NodePort" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.dev.enabled=true' \
      --set 'server.service.nodePort=30008' \
      . | tee /dev/stderr |
      yq -r '.spec.ports[0].nodePort' | tee /dev/stderr)
  [ "${actual}" = "null" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.ha.enabled=true' \
      --set 'server.service.nodePort=30009' \
      . | tee /dev/stderr |
      yq -r '.spec.ports[0].nodePort' | tee /dev/stderr)
  [ "${actual}" = "null" ]

  local actual=$(helm template \
      --show-only templates/server-service.yaml \
      --set 'server.standalone.enabled=true' \
      --set 'server.service.nodePort=30010' \
      . | tee /dev/stderr |
      yq -r '.spec.ports[0].nodePort' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}
