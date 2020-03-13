#!/usr/bin/env bats

load _helpers

@test "server/ingress: disabled by default" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-ingress.yaml  \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/ingress: disable by injector.externalVaultAddr" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/server-ingress.yaml  \
      --set 'server.ingress.enabled=true' \
      --set 'injector.externalVaultAddr=http://vault-outside' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "server/ingress: checking host entry gets added and path is /" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/server-ingress.yaml \
      --set 'server.ingress.enabled=true' \
      --set 'server.ingress.hosts[0].host=test.com' \
      --set 'server.ingress.hosts[0].paths[0]=/' \
      . | tee /dev/stderr |
      yq  -r '.spec.rules[0].host' | tee /dev/stderr)
  [ "${actual}" = 'test.com' ]

  local actual=$(helm template \
      --show-only templates/server-ingress.yaml \
      --set 'server.ingress.enabled=true' \
      --set 'server.ingress.hosts[0].host=test.com' \
      --set 'server.ingress.hosts[0].paths[0]=/' \
      . | tee /dev/stderr |
      yq  -r '.spec.rules[0].http.paths[0].path' | tee /dev/stderr)
  [ "${actual}" = '/' ]
}

@test "server/ingress: vault backend should be added when I specify a path" {
  cd `chart_dir`

  local actual=$(helm template \
      --show-only templates/server-ingress.yaml \
      --set 'server.ingress.enabled=true' \
      --set 'server.ingress.hosts[0].host=test.com' \
      --set 'server.ingress.hosts[0].paths[0]=/' \
      . | tee /dev/stderr |
      yq  -r '.spec.rules[0].http.paths[0].backend.serviceName  | length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]

}

@test "server/ingress: labels gets added to object" {
  cd `chart_dir`

  local actual=$(helm template \
      --show-only templates/server-ingress.yaml \
      --set 'server.ingress.enabled=true' \
      --set 'server.ingress.labels.traffic=external' \
      --set 'server.ingress.labels.team=dev' \
      . | tee /dev/stderr |
      yq -r '.metadata.labels.traffic' | tee /dev/stderr)
  [ "${actual}" = "external" ]
}
