/*
 * Google Container Registry: https://www.terraform.io/docs/providers/google/r/container_registry.html
 * Private container registry to push and pull proprietary pod images
 */
resource "google_container_registry" "registry" {
  // No arguments are needed, `project` is inherited from provider
}

/*
 * Google Service Account: https://www.terraform.io/docs/providers/google/r/google_service_account.html
 * Creation of the service account to provide write access to GCR
 */
resource "google_service_account" "gcr" {
  account_id   = "gcrservice"
  display_name = "GCR Service Account"
  description  = "Service account to provide GCR write access to build pipelines"
}

/*
 * Google IAM Policy: https://www.terraform.io/docs/providers/google/d/iam_policy.html
 * Policy to allow read and write access to GCR
 */
data "google_iam_policy" "gcr" {
  binding {
    role = "roles/storage.objectAdmin"
    members = [
      "serviceAccount:${google_service_account.gcr.email}"
    ]
  }
}

/*
 * Google Service Account Policy: https://www.terraform.io/docs/providers/google/r/google_service_account_iam.html
 * Binds the GCR read/write IAM policy to the appropriate service account
 */
resource "google_service_account_iam_policy" "gcr" {
  service_account_id = google_service_account.gcr.name
  policy_data        = data.google_iam_policy.gcr.policy_data
}

/*
 * Google Service Account Key: https://www.terraform.io/docs/providers/google/r/google_service_account_key.html
 * Create the service account credentials for use within the build pipelines to push to GCR
 */
resource "google_service_account_key" "gcr" {
  service_account_id = google_service_account.gcr.name
}

/*
 * Local File: https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file
 * Renders the private key JSON for the GCR service account to a local file to be uploaded to the build pipeline environment
 */
resource "local_file" "gcr_svcacc_key" {
  content  = base64decode(google_service_account_key.gcr.private_key)
  filename = "${path.module}/gcr_account_key.json"
}
