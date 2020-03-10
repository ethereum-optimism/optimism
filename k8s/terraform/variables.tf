variable "k8s_config_path" {
  type        = string
  default     = "~/.kube/config"
  description = "Path to the local Kubernetes configuration file"
}

variable "k8s_context_cluster" {
  type        = string
  default     = "minikube"
  description = "Name of the context from the Kubernetes configuration file for the cluster to target"
}

variable "consul_gossip_key_name" {
  type        = string
  default     = "consul-gossip-encryption-key"
  description = "The name of the secret in Kubernetes to store the Consul gossip key"
}

variable "consul_datacenter" {
  type        = string
  default     = "dc1"
  description = "The datacenter to create in the Consul cluster"
}

variable "consul_replicas" {
  type        = number
  default     = 5
  description = "The number of Consul servers to create in the cluster"
}

variable "consul_bootstrap_expect" {
  type        = number
  default     = 3
  description = "Number of Consul nodes to expect for bootstrapping the cluster"
}

variable "vault_addr" {
  type        = string
  default     = "https://localhost:8200"
  description = "The address to the Vault server for the provider to utilize"
}
