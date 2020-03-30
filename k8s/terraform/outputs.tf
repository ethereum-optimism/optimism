output "helm_consul_status" {
  value = helm_release.consul_chart.status
}

output "helm_vault_status" {
  value = helm_release.vault_chart.status
}
