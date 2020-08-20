/*
 * KMS Key Ring: https://www.terraform.io/docs/providers/google/r/kms_key_ring.html
 * Create the KMS key ring to hold the autounseal key for Vault
 */
resource "google_kms_key_ring" "vault" {
  name     = "omgnetwork-vault-ring"
  location = "global"
}

/*
 * KMS Crypto Key: https://www.terraform.io/docs/providers/google/r/kms_crypto_key.html
 * Create the KMS key on the Vault key ring used for unsealing
 */
resource "google_kms_crypto_key" "unseal" {
  name            = "omgnetwork-vault-unseal-key"
  key_ring        = google_kms_key_ring.vault.id
  rotation_period = "86400s"
}

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
resource "google_project_iam_binding" "kms" {
  role = "roles/cloudkms.admin"
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
