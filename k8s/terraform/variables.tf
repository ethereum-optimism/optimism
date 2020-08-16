variable "docker_registry_host" {
  type        = string
  default     = "gcr.io"
  description = "The host of the Docker registry to use for pulling pod images, gets combined with `gcp_project` to make registry addr"
}

variable "gcp_project" {
  type        = string
  description = "Name of the GCP project being targeted if deploying"
}

variable "gcp_region" {
  type        = string
  description = "GCP region that the target project is in"
}

variable "gke_cluster_name" {
  type        = string
  description = "Name of the GKE cluster created by the infrastructure scripts"
}

variable "k8s_certificates_secret_name_prefix" {
  type        = string
  default     = "omisego-certificates"
  description = "The name of the secret in Kubernetes storing the certificates for the cluster servers"
}

variable "k8s_config_path" {
  type        = string
  default     = "~/.kube/config"
  description = "Path to the local Kubernetes configuration file"
}

variable "k8s_namespace" {
  type        = string
  default     = "default"
  description = "Kubernetes namespace to install the Helm charts into"
}

variable "local_certificates_dir" {
  type        = string
  description = "Absolute path to the directory storing the generated cluster service certificates"
}

variable "recovery" {
  type        = bool
  default     = false
  description = "Recovering from a disaster"
}

variable "unsealer_vault_addr" {
  type        = string
  default     = "https://10.8.0.2:8200"
  description = "The address to the Unsealer Vault server"
}

variable "vault_replicas" {
  type        = number
  default     = 3
  description = "The number of Vault server pods to run in the cluster"
}
