terraform {
  required_version = ">= 0.12"
}

/*
 * Kubernetes Provider - https://www.terraform.io/docs/providers/kubernetes/guides/getting-started.html
 * Kubernetes configuration for specifying the
 * cluster target context and interaction with the cluster
 * secrets API
 */
provider "kubernetes" {
  config_path            = var.k8s_config_path
  config_context_cluster = var.k8s_context_cluster
}

/*
 * Helm Provider - https://www.terraform.io/docs/providers/helm/
 * Helm configuration for installing charts for
 * Consul and Vault within the target K8S cluster
 */
provider "helm" {
  kubernetes {
    config_path    = var.k8s_config_path
    config_context = var.k8s_context_cluster
  }
}

/*
 * Vault Provider - https://www.terraform.io/docs/providers/vault/
 * Vault configuration for utilizing unsealer instance
 * secrets and managing the lifecycle of K8S secrets storage
 */
provider "vault" {
  address = var.unsealer_vault_addr
}
