output "helm_consul_status" {
  value = helm_release.consul_chart.status
}

output "helm_vault_status" {
  value = helm_release.vault_chart.status
}

output "disaster_recovery_steps" {
  value = <<-EOT
    In the event of disaster recovery, perform the followings steps:
    (1) Delete the existing local Terraform state file:
      `rm *.tfstate*`
    (2) Destroy the Vault and Consul Kubernetes resources:
      `kubectl -n ${var.k8s_namespace} delete pod,svc,deployment,statefulset,secret --all`
    (3) Change the Terraform variable `recovery` to true in your `.tfvars` file
    (4) Re-apply the Terraform script:
      `terraform plan`
      `terraform apply`
  EOT
}
