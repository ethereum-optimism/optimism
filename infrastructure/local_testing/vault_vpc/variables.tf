variable "gcp_project" {
  description = "Name of GCP project used to provision infrastructure into"
}

variable "gcp_region" {
  description = "GCP region to provision resources into"
}

variable "gcp_zone" {
  description = "GCP zone to provision resources into"
}

variable "datadog_api_key" {
  description = "API key used by Datadog agent in the instance to authenticate to Datadog"
}

variable "ssh_user" {
  description = "Email address of the user with access to SSH into the instance"
}

variable "vault_vpc_name" {
  description = "Name of the VPC"
}

variable "vault_vpc_uri" {
  description = "URI of the VPC"
}

variable "vault_vpc_subnet" {
  description = "Vault VPC's subnet"
}
