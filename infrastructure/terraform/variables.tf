variable "datadog_api_key" {
  description = "Datadog account API key"
  type        = string
}

variable "datadog_app_key" {
  description = "Datadog application key"
  type        = string
}

variable "gcp_project" {
  description = "Name of GCP project used to provision infrastructure into"
  type        = string
}

variable "gcp_region" {
  description = "GCP region to provision resources into"
  type        = string
}

variable "gke_cluster_name" {
  description = "Name of the GKE Kubernetes cluster to create"
  type        = string
}

variable "gke_node_count" {
  description = "The number of nodes to create in the GKE node pool"
  default     = 3
  type        = number
}

variable "gke_pod_cidr" {
  description = "CIDR block for the Vault K8S pods to be in (should be /21 or lower block)"
  type        = string
}

variable "gke_service_cidr" {
  description = "CIDR block for the Vault K8S service to be in"
  type        = string
}

variable "omgnetwork_cidrs" {
  description = "List of CIDR blocks used when allowing ingress access in Vault VPC firewall"
  type        = list(string)
}

variable "omgnetwork_vpc_uri" {
  description = "URI of the client VPC to be peered to the Vault VPC"
  type        = string
}

variable "router_asn" {
  description = "ASN used for the router. Needs to be a valid ASN number not use elsewhere"
  default     = 64512
  type        = number
}

variable "vault_subnet_cidr" {
  description = "The subnet that the Vault cluster should be deployed under in the VPC"
  type        = string
}
