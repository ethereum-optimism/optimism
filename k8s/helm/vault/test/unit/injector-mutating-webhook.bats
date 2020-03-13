#!/usr/bin/env bats

load _helpers

@test "injector/MutatingWebhookConfiguration: enabled by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/injector-mutating-webhook.yaml  \
      . | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "true" ]
}

@test "injector/MutatingWebhookConfiguration: disable with global.enabled false" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/injector-mutating-webhook.yaml  \
      --set 'global.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "injector/MutatingWebhookConfiguration: disable with injector.enabled false" {
  cd `chart_dir`
  local actual=$( (helm template \
      --show-only templates/injector-mutating-webhook.yaml  \
      --set 'injector.enabled=false' \
      . || echo "---") | tee /dev/stderr |
      yq 'length > 0' | tee /dev/stderr)
  [ "${actual}" = "false" ]
}

@test "injector/MutatingWebhookConfiguration: namespace is set" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/injector-mutating-webhook.yaml  \
      --set 'injector.enabled=true' \
      --namespace foo \
      . | tee /dev/stderr |
      yq '.webhooks[0].clientConfig.service.namespace' | tee /dev/stderr)
  [ "${actual}" = "\"foo\"" ]
}

@test "injector/MutatingWebhookConfiguration: caBundle is empty" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/injector-mutating-webhook.yaml  \
      --set 'injector.enabled=true' \
      --namespace foo \
      . | tee /dev/stderr |
      yq '.webhooks[0].clientConfig.caBundle' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

@test "injector/MutatingWebhookConfiguration: namespaceSelector empty by default" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/injector-mutating-webhook.yaml  \
      --set 'injector.enabled=true' \
      --namespace foo \
      . | tee /dev/stderr |
      yq '.webhooks[0].namespaceSelector' | tee /dev/stderr)
  [ "${actual}" = "null" ]
}

@test "injector/MutatingWebhookConfiguration: can set namespaceSelector" {
  cd `chart_dir`
  local actual=$(helm template \
      --show-only templates/injector-mutating-webhook.yaml  \
      --set 'injector.enabled=true' \
      --set 'injector.namespaceSelector.matchLabels.injector=true' \
      . | tee /dev/stderr |
      yq '.webhooks[0].namespaceSelector.matchLabels.injector' | tee /dev/stderr)

  [ "${actual}" = "true" ]
}
