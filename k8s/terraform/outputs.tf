output "helm_consul_status" {
  value = helm_release.consul_chart.status
}

output "helm_vault_status" {
  value = helm_release.vault_chart.status
}

output "vault_service_data" {
  value = jsonencode(data.kubernetes_service.vault)
}
