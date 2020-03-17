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

variable "k8s_consul_bootstrap_acl_token_secret_name" {
  type        = string
  default     = "omisego-consul-bootstrap-acl-token"
  description = "The name of the Kubernetes secret for storing the bootstrap ACL token"
}

variable "k8s_consul_gossip_secret_name" {
  type        = string
  default     = "omisego-consul-gossip-encryption-key"
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

variable "unsealer_vault_addr" {
  type        = string
  default     = "https://localhost:8200"
  description = "The address to the Unsealer Vault server"
}

variable "unsealer_vault_token" {
  type        = string
  description = "The Vault token to be used for unsealing the Vault cluster"
}
