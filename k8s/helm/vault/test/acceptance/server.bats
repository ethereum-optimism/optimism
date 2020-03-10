#!/usr/bin/env bats

load _helpers

@test "server: default, comes up sealed" {
  helm_install
  wait_for_running $(name_prefix)-server-0

  # Verify installed, sealed, and 1 replica
  local sealed_status=$(kubectl exec "$(name_prefix)-server-0" -- vault status -format=json | 
    jq .sealed )
  [ "${sealed_status}" == "true" ]

  local init_status=$(kubectl exec "$(name_prefix)-server-0" -- vault status -format=json | 
    jq .initialized)
  [ "${init_status}" == "false" ]

  # TODO check pv, pvc
}

# Clean up
teardown() {
  echo "helm/pvc teardown"
  helm delete --purge vault
  kubectl delete --all pvc 
}
