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
 *  Data Dog Provider - https://www.terraform.io/docs/providers/datadog/index.html
 *  Used to retrieve Data Dog's IP addresses in order to configure egreess firewall rules
 *  The provider requires the DATADOG_API_KEY and DATADOG_APP_KEY environment variables
 */

provider "datadog" {}