/*
 * Google DNS Managed Zone: https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_managed_zone
 * For the name of the ingress to the Vault cluster, and for DNS-01 ACME challenges for SSL certificates.
 */

resource "google_dns_managed_zone" "vault" {
  name        = "vault"
  dns_name    = var.vault_dns_zone
}

resource "google_dns_record_set" "vault_ingress" {
  name = var.vault_ingress_fqdn
  type = "A"
  ttl  = 300

  managed_zone = google_dns_managed_zone.vault.name

  rrdatas = [var.vault_ingress_ip]
}

resource "google_service_account" "dns01_solver" {
  account_id   = "dns01-solver"
  display_name = "DNS01 Solver Service Account"
  description  = "Service account to solve ACME challenges for certificates via DNS01"
}

resource "google_project_iam_binding" "dns_admin" {
  role = "roles/dns.admin"
  members = [
    "serviceAccount:${google_service_account.dns01_solver.email}"
  ]
}

resource "google_service_account_key" "dns01-solver" {
  service_account_id = google_service_account.dns01_solver.name
}

/*
 * Local File: https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file
 * Renders the private key JSON for the GCR service account to a local file to be uploaded to the build pipeline environment
 */
resource "local_file" "dns_svcacc_key" {
  content  = base64decode(google_service_account_key.dns01-solver.private_key)
  filename = "${path.module}/dns_account.key.json"
}
