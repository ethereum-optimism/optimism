/*
 * Google Service Account: https://www.terraform.io/docs/providers/google/r/google_service_account.html
 * Creation of the service account to provide access to KMS
 */
resource "google_service_account" "kms" {
  account_id   = "kmsservice"
  display_name = "KMS Service Account"
  description  = "Service account to provide access to KMS"
}

/*
 * Google IAM Binding: https://www.terraform.io/docs/providers/google/r/google_project_iam.html
 * Assign the custom IAM role to the KMS service account
 */
resource "google_project_iam_binding" "kms_admin" {
  role = "roles/cloudkms.admin"
  members = [
    "serviceAccount:${google_service_account.kms.email}"
  ]
}

/*
 * Google IAM Binding: https://www.terraform.io/docs/providers/google/r/google_project_iam.html
 * Assign the custom IAM role to the KMS service account
 */
resource "google_project_iam_binding" "kms_use" {
  role = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  members = [
    "serviceAccount:${google_service_account.kms.email}"
  ]
}

/*
 * Google Service Account Key: https://www.terraform.io/docs/providers/google/r/google_service_account_key.html
 * Create the service account credentials for autounseal in Vault using KMS
 */
resource "google_service_account_key" "kms" {
  service_account_id = google_service_account.kms.name
}

/*
 * Local File: https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file
 * Renders the private key JSON for the GCR service account to a local file to be uploaded to the build pipeline environment
 */
resource "local_file" "kms_svcacc_key" {
  content  = base64decode(google_service_account_key.kms.private_key)
  filename = "${path.module}/kms_account.key.json"
}
