#!/usr/bin/env bats

load _helpers

@test "injector: testing deployment" {
  cd `chart_dir`
  
  kubectl delete namespace acceptance --ignore-not-found=true
  kubectl create namespace acceptance
  kubectl config set-context --current --namespace=acceptance

  kubectl create -f ./test/acceptance/injector-test/pg-deployment.yaml
  sleep 5
  wait_for_ready $(kubectl get pod -l app=postgres -o jsonpath="{.items[0].metadata.name}")

  kubectl create secret generic test \
    --from-file ./test/acceptance/injector-test/pgdump-policy.hcl \
    --from-file ./test/acceptance/injector-test/bootstrap.sh 

  kubectl label secret test app=vault-agent-demo

  helm install "$(name_prefix)" \
    --set="server.extraVolumes[0].type=secret" \
    --set="server.extraVolumes[0].name=test" .
  wait_for_running $(name_prefix)-0

  wait_for_ready $(kubectl get pod -l component=webhook -o jsonpath="{.items[0].metadata.name}")

  kubectl exec -ti "$(name_prefix)-0" -- /bin/sh -c "cp /vault/userconfig/test/bootstrap.sh /tmp/bootstrap.sh && chmod +x /tmp/bootstrap.sh && /tmp/bootstrap.sh"
  sleep 5

    # Sealed, not initialized
  local sealed_status=$(kubectl exec "$(name_prefix)-0" -- vault status -format=json |
    jq -r '.sealed' )
  [ "${sealed_status}" == "false" ]

  local init_status=$(kubectl exec "$(name_prefix)-0" -- vault status -format=json |
    jq -r '.initialized')
  [ "${init_status}" == "true" ] 


  kubectl create -f ./test/acceptance/injector-test/job.yaml
  wait_for_complete_job "pgdump"
}

# Clean up
teardown() {
  echo "helm/pvc teardown"
  helm delete vault
  kubectl delete --all pvc
  kubectl delete secret test 
  kubectl delete job pgdump
  kubectl delete deployment postgres
  kubectl delete namespace acceptance
}
