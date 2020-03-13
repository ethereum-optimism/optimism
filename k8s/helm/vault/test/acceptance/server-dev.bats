#!/usr/bin/env bats

load _helpers

@test "server/dev: testing deployment" {
  cd `chart_dir`
  kubectl delete namespace acceptance --ignore-not-found=true
  kubectl create namespace acceptance
  kubectl config set-context --current --namespace=acceptance

  helm install "$(name_prefix)" --set='server.dev.enabled=true' .
  wait_for_running $(name_prefix)-0

  # Replicas
  local replicas=$(kubectl get statefulset "$(name_prefix)" --output json |
    jq -r '.spec.replicas')
  [ "${replicas}" == "1" ]

  # Volume Mounts
  local volumeCount=$(kubectl get statefulset "$(name_prefix)" --output json |
    jq -r '.spec.template.spec.containers[0].volumeMounts | length')
  [ "${volumeCount}" == "0" ]

  # Service
  local service=$(kubectl get service "$(name_prefix)" --output json |
    jq -r '.spec.clusterIP')
  [ "${service}" != "None" ]

  local service=$(kubectl get service "$(name_prefix)" --output json |
    jq -r '.spec.type')
  [ "${service}" == "ClusterIP" ]

  local ports=$(kubectl get service "$(name_prefix)" --output json |
    jq -r '.spec.ports | length')
  [ "${ports}" == "2" ]

  local ports=$(kubectl get service "$(name_prefix)" --output json |
    jq -r '.spec.ports[0].port')
  [ "${ports}" == "8200" ]

  local ports=$(kubectl get service "$(name_prefix)" --output json |
    jq -r '.spec.ports[1].port')
  [ "${ports}" == "8201" ]

  # Sealed, not initialized
  local sealed_status=$(kubectl exec "$(name_prefix)-0" -- vault status -format=json |
    jq -r '.sealed' )
  [ "${sealed_status}" == "false" ]

  local init_status=$(kubectl exec "$(name_prefix)-0" -- vault status -format=json |
    jq -r '.initialized')
  [ "${init_status}" == "true" ]
}

# Clean up
teardown() {
  echo "helm/pvc teardown"
  helm delete vault
  kubectl delete --all pvc
  kubectl delete namespace acceptance --ignore-not-found=true
}
