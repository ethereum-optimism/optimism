variable "gcp_project_omisego" {
  description = "Name of GCP project used for represent Omisego VPC"
}

variable "gcp_region" {
  description = "GCP region to provision resources into"
}

variable "gcp_zone" {
  description = "GCP zone to provision resources into"
}

variable "ssh_user" {
  description = "Email address of the user with access to SSH into the instance"
}

variable "vault_vpc_uri" {
  description = "URI of the VPC"
}

variable "omisego_subnet_cidr" {
  description = "CIDR block for Omisego subnet"
}