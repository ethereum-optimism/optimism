/*
 * Kubernetes Provider - https://www.terraform.io/docs/providers/kubernetes/guides/getting-started.html
 * Kubernetes configuration for specifying the
 * cluster target context and interaction with the cluster
 * secrets API
 */
provider "kubernetes" {
  config_path            = var.k8s_config_path
  config_context_cluster = "gke_${var.gcp_project}_${var.gcp_region}_${var.gke_cluster_name}"
}

/*
 * Vault Provider - https://www.terraform.io/docs/providers/vault/
 * Vault configuration for utilizing unsealer instance
 * secrets and managing the lifecycle of K8S secrets storage
 */
provider "vault" {
  address = var.unsealer_vault_addr
}

/*
 * Setup local variables to leverage in the rest of the scripts
 */
locals {
  image_registry = "${trimsuffix(var.docker_registry_host, "/")}/${var.gcp_project}"
  vault_img      = "${local.image_registry}/omgnetwork/vault:1.0.0"
}
