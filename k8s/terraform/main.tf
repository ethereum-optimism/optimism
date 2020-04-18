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
  config_context_cluster = "gke_${var.gcp_project}_${var.gcp_region}_${var.gke_cluster_name}"
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

/*
 * Setup local variables to leverage in the rest of the scripts
 */
locals {
  image_registry = "${trimsuffix(var.docker_registry_host, "/")}/${var.gcp_project}"
  consul_img     = "${local.image_registry}/consul:1.7.1"
  consul_k8s_img = "${local.image_registry}/hashicorp/consul-k8s:0.12.0"
  vault_img      = "${local.image_registry}/omisego/immutability-vault-ethereum:1.0.0"
}
