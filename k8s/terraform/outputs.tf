output "helm_consul_status" {
  value = helm_release.consul_chart.status
}

output "helm_vault_status" {
  value = helm_release.vault_chart.status
}

output "vault_service_data" {
  value = jsonencode(data.kubernetes_service.vault)
}

output "vault_service_ip" {
  value = data.kubernetes_service.vault.spec[0].type == "ClusterIP" ? data.kubernetes_service.vault.spec[0].cluster_ip : data.kubernetes_service.vault.spec[0].load_balancer_ip
}
