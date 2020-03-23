variable "consul_bootstrap_expect" {
  type        = number
  default     = 3
  description = "Number of Consul nodes to expect for bootstrapping the cluster"
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

variable "k8s_context_cluster" {
  type        = string
  default     = "minikube"
  description = "Name of the context from the Kubernetes configuration file for the cluster to target"
}

variable "k8s_consul_bootstrap_acl_token_name" {
  type        = string
  default     = "omisego-consul-bootstrap-acl-token"
  description = "The name of the Kubernetes secret for storing the bootstrap ACL token"
}

variable "k8s_consul_client_acl_token_name" {
  type        = string
  default     = "omisego-consul-client-acl-token"
  description = "The name of the Kubernetes secret that will have the Consul client ACL token to clean"
}

variable "k8s_consul_vault_acl_token_name" {
  type        = string
  default     = "omisego-consul-vault-acl-token"
  description = "The name of the Kubernetes secret that will have the Consul Vault ACL token to clean"
}

variable "k8s_consul_gossip_key_name" {
  type        = string
  default     = "omisego-consul-gossip-encryption-key"
  description = "The name of the secret in Kubernetes to store the Consul gossip key"
}

variable "local_certificates_dir" {
  type        = string
  description = "Absolute path to the directory storing the generated cluster service certificates"
}

variable "tls_enabled" {
  type        = bool
  default     = false
  description = "Whether to enable TLS communication within the services"
}

variable "unsealer_vault_addr" {
  type        = string
  default     = "https://127.0.0.1:8200"
  description = "The address to the Unsealer Vault server"
}

variable "vault_replicas" {
  type        = number
  default     = 3
  description = "The number of Vault server pods to run in the cluster"
}

