#!/usr/bin/env bats

load _helpers

@test "server/standalone: testing deployment" {
  cd `chart_dir`

  kubectl delete namespace acceptance --ignore-not-found=true
  kubectl create namespace acceptance
  kubectl config set-context --current --namespace=acceptance

  helm install "$(name_prefix)" .
  wait_for_running $(name_prefix)-0

  # Sealed, not initialized
  local sealed_status=$(kubectl exec "$(name_prefix)-0" -- vault status -format=json |
    jq -r '.sealed' )
  [ "${sealed_status}" == "true" ]

  local init_status=$(kubectl exec "$(name_prefix)-0" -- vault status -format=json |
    jq -r '.initialized')
  [ "${init_status}" == "false" ]

  # Security
  local ipc=$(kubectl get statefulset "$(name_prefix)" --output json |
    jq -r '.spec.template.spec.containers[0].securityContext.capabilities.add[0]')
  [ "${ipc}" == "IPC_LOCK" ]

  # Replicas
  local replicas=$(kubectl get statefulset "$(name_prefix)" --output json |
    jq -r '.spec.replicas')
  [ "${replicas}" == "1" ]

  # Affinity
  local affinity=$(kubectl get statefulset "$(name_prefix)" --output json |
    jq -r '.spec.template.spec.affinity')
  [ "${affinity}" != "null" ]

  # Volume Mounts
  local volumeCount=$(kubectl get statefulset "$(name_prefix)" --output json |
    jq -r '.spec.template.spec.containers[0].volumeMounts | length')
  [ "${volumeCount}" == "2" ]

  local mountName=$(kubectl get statefulset "$(name_prefix)" --output json |
    jq -r '.spec.template.spec.containers[0].volumeMounts[0].name')
  [ "${mountName}" == "data" ]

  local mountPath=$(kubectl get statefulset "$(name_prefix)" --output json |
    jq -r '.spec.template.spec.containers[0].volumeMounts[0].mountPath')
  [ "${mountPath}" == "/vault/data" ]

  # Volumes
  local volumeCount=$(kubectl get statefulset "$(name_prefix)" --output json |
    jq -r '.spec.template.spec.volumes | length')
  [ "${volumeCount}" == "1" ]

  local volume=$(kubectl get statefulset "$(name_prefix)" --output json |
    jq -r '.spec.template.spec.volumes[0].configMap.name')
  [ "${volume}" == "$(name_prefix)-config" ]

  # Security Context
  local fsGroup=$(kubectl get statefulset "$(name_prefix)" --output json |
    jq -r '.spec.template.spec.securityContext.fsGroup')
  [ "${fsGroup}" == "1000" ]

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

  # Vault Init
  local token=$(kubectl exec -ti "$(name_prefix)-0" -- \
    vault operator init -format=json -n 1 -t 1 | \
    jq -r '.unseal_keys_b64[0]')
  [ "${token}" != "" ]

  # Vault Unseal
  local pods=($(kubectl get pods --selector='app.kubernetes.io/name=vault' -o json | jq -r '.items[].metadata.name'))
  for pod in "${pods[@]}"
  do
      kubectl exec -ti ${pod} -- vault operator unseal ${token}
  done

  wait_for_ready "$(name_prefix)-0"

  # Unsealed, initialized
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
