variable "gcp_project" {
  description = "Name of GCP project used for represent OMG Network VPC"
}

variable "gcp_region" {
  description = "GCP region to provision resources into"
}

variable "gcp_zone" {
  description = "GCP zone to provision resources into"
}

variable "vault_vpc_uri" {
  description = "URI of the VPC"
}

variable "subnet_cidr" {
  description = "CIDR block for OMG Network subnet"
}

variable "ssh_user" {
  description = "Email address of the user with access to SSH into the instance"
}

variable "ssh_cidr_list" {
  description = "CIDR block for OMG Network subnet"
  type        = list(string)
}
