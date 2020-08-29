variable "gcp_project" {
  description = "Name of GCP project used to provision infrastructure into"
  type        = string
}

variable "gcp_region" {
  description = "GCP region to provision resources into"
  type        = string
}

variable "subnet_cidr" {
  description = "Subnet used to provision resources into"
  type        = string
}

variable "router_asn" {
  description = "ASN used for the router. Needs to be a valid ASN number not use elsewhere"
  default     = 64512
  type        = number
}

variable "datadog_api_key" {
  description = "Datadog API key"
  type        = string
}

variable "datadog_app_key" {
  description = "Datadog APP key"
  type        = string
}

variable "omgnetwork_vpc_uri" {
  description = "URI of the client VPC to be peered to the Vault VPC"
  type        = string
}

variable "omgnetwork_subnet_cidr" {
  description = "CIDR block of subnet used when allowing ingress access in Vault VPC firewall"
  type        = string
}

variable "lockdown_egress" {
  description = "Boolean indicating if egress network access is lockdown to only Datadog IPs"
  default     = false
  type        = bool
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