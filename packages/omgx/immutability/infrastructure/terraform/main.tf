/*
 * GCP Provider - https://www.terraform.io/docs/providers/google/guides/provider_reference.html
 * Required for provisioning infrastructure in GCP
 */
provider "google" {
  project = var.gcp_project
  region  = var.gcp_region
  batching {
    enable_batching = false
  }
}

provider "google-beta" {
  project = var.gcp_project
  region  = var.gcp_region
  batching {
    enable_batching = false
  }
}

provider "datadog" {
  api_key = var.datadog_api_key
  app_key = var.datadog_app_key
}
