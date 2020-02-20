variable "gcp_project" {
  description = "Name of GCP project used to provision infrastructure into"
}

variable "gcp_region" {
  description = "GCP region to provision resources into"
}

variable "subnet_cidr" {
  description = "Subnet used to provision resources into"
}

variable "router_asn" {
  description = "ASN used for the router. Needs to be a valid ASN number not use elsewhere"
  default     = 64512
}