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
 * Google Custom IAM Role: https://www.terraform.io/docs/providers/google/r/google_project_iam_custom_role.html
 * Create a customized role for GCR access with a service account
 */
resource "google_project_iam_custom_role" "gcr_admin" {
  role_id     = "gcrAdminRole"
  title       = "GCR Admin Role"
  description = "A custom role to provide custom read/write access for GCR management"
  permissions = [
    "storage.buckets.create",
    "storage.buckets.delete",
    "storage.buckets.get",
    "storage.buckets.list",
    "storage.buckets.update",
    "storage.objects.create",
    "storage.objects.delete",
    "storage.objects.get",
    "storage.objects.list",
    "storage.objects.update"
  ]
}

/*
 * Google IAM Binding: https://www.terraform.io/docs/providers/google/r/google_project_iam.html
 * Assign the custom IAM role to the GCR service account
 */
resource "google_project_iam_binding" "gcr" {
  role = google_project_iam_custom_role.gcr_admin.id
  members = [
    "serviceAccount:${google_service_account.gcr.email}"
  ]
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
  filename = "${path.module}/gcr_account.key.json"
}
