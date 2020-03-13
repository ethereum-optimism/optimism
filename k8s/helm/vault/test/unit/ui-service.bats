#!/usr/bin/env bats

load _helpers

@test "ui/Service: disabled by default" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/ui-service.yaml \
      --set 'server.dev.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.ha.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.standalone.enabled=true' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "ui/Service: disable with ui.enabled" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'ui.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'ui.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.standalone.enabled=true' \
      --set 'ui.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "ui/Service: disable with injector.externalVaultAddr" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]

  local actual=$( (helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.standalone.enabled=true' \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "ui/Service: ClusterIP type by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'ui.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "ClusterIP" ]

  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'ui.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "ClusterIP" ]

  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.standalone.enabled=true' \
      --set 'ui.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "ClusterIP" ]
}

@test "ui/Service: specified type" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'ui.serviceType=LoadBalancer' \
      --set 'ui.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "LoadBalancer" ]

  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'ui.serviceType=LoadBalancer' \
      --set 'ui.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "LoadBalancer" ]

  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.standalone.enabled=true' \
      --set 'ui.serviceType=LoadBalancer' \
      --set 'ui.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.spec.type' | tee /dev/stderr)
  [ "${actual}" = "LoadBalancer" ]
}

@test "ui/Service: LoadBalancerIP set if specified and serviceType == LoadBalancer" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'ui.serviceType=LoadBalancer' \
      --set 'ui.enabled=true' \
      --set 'ui.loadBalancerIP=123.123.123.123' \
      . | tee /dev/stderr |
      yq -r '.spec.loadBalancerIP' | tee /dev/stderr)
  [ "${actual}" = "123.123.123.123" ]

  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'ui.serviceType=ClusterIP' \
      --set 'ui.enabled=true' \
      --set 'ui.loadBalancerIP=123.123.123.123' \
      . | tee /dev/stderr |
      yq -r '.spec.loadBalancerIP' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

@test "ui/Service: set loadBalancerSourceRanges when LoadBalancer is configured as serviceType" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'ui.serviceType=LoadBalancer' \
      --set 'ui.enabled=true' \
      --set 'ui.loadBalancerSourceRanges={"123.123.123.123"}' \
      . | tee /dev/stderr |
      yq -r '.spec.loadBalancerSourceRanges[0]' | tee /dev/stderr)
  [ "${actual}" = "123.123.123.123" ]

  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'ui.serviceType=ClusterIP' \
      --set 'ui.enabled=true' \
      --set 'ui.loadBalancerSourceRanges={"123.123.123.123"}' \
      . | tee /dev/stderr |
      yq -r '.spec.loadBalancerSourceRanges[0]' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

@test "ui/Service: specify annotations" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.dev.enabled=true' \
      --set 'ui.serviceType=LoadBalancer' \
      --set 'ui.enabled=true' \
      --set 'ui.annotations.foo=bar' \
      . | tee /dev/stderr |
      yq -r '.metadata.annotations["foo"]' | tee /dev/stderr)
  [ "${actual}" = "bar" ]

  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'ui.serviceType=LoadBalancer' \
      --set 'ui.enabled=true' \
      --set 'ui.annotations.foo=bar' \
      . | tee /dev/stderr |
      yq -r '.metadata.annotations["foo"]' | tee /dev/stderr)
  [ "${actual}" = "bar" ]

  local actual=$(helm template \
      --show-only templates/ui-service.yaml  \
      --set 'server.ha.enabled=true' \
      --set 'ui.serviceType=LoadBalancer' \
      --set 'ui.enabled=true' \
      . | tee /dev/stderr |
      yq -r '.metadata.annotations["foo"]' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}
