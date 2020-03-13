variable "project" {
  default = "vault-helm-dev-246514"

  description = <<EOF
Google Cloud Project to launch resources in. This project must have GKE
enabled and billing activated. We can't use the GOOGLE_PROJECT environment
variable since we need to access the project for other uses.
EOF
}

variable "zone" {
  default     = "us-central1-a"
  description = "The zone to launch all the GKE nodes in."
}

variable "init_cli" {
  default     = true
  description = "Whether to init kubectl or not."
}

variable "gcp_service_account" {
  default = "vault-terraform-helm-test"

  description = <<EOF
Service account used on the nodes to manage/use the API, specifically needed
for using auto-unseal
EOF
}
