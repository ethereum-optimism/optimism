variable "datadog_api_key" {
  description = "API key used by Datadog agent in the instance to authenticate to Datadog"
}

variable "ssh_user" {
  description = "Email address of the user with access to SSH into the instance"
}

variable "vault_vpc_name" {
  description = "Name of the VPC"
}

variable "vault_vpc_subnet" {
  description = "Vault VPC's subnet"
}