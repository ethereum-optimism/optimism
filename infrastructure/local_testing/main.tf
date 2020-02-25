/*
 * GCP Provider - https://www.terraform.io/docs/providers/google/guides/provider_reference.html
 * Required for provisioning infrastructure in GCP
 */
provider "google" {
  project = var.gcp_project
  region  = var.gcp_region
}

provider "google-beta" {
  project = var.gcp_project
  region  = var.gcp_region
}

provider "google" {
  alias   = "omisego"
  project = var.gcp_project_omisego
  region  = var.gcp_region
}

provider "google-beta" {
  alias   = "omisego-beta"
  project = var.gcp_project_omisego
  region  = var.gcp_region
}

