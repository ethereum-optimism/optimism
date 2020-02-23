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

/*
 *  Datadog Provider - https://www.terraform.io/docs/providers/datadog/index.html
 *  Used to retrieve Datadog's IP addresses in order to configure egreess firewall rules
 *  The provider requires the DATADOG_API_KEY and DATADOG_APP_KEY environment variables
 */

provider "datadog" {
  api_key = var.datadog_api_key
  app_key = var.datadog_app_key
}