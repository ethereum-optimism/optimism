/*
 * GPC Zones - https://www.terraform.io/docs/providers/google/d/google_compute_zones.html
 * Used to select the zone where the instance is going to be provissioned
 */
data "google_compute_zones" "available" {
  region = var.gcp_region
}

/*
 * IAP Tunnel - https://www.terraform.io/docs/providers/google/r/iap_tunnel_instance_iam.html
 * An IAP tunnel grants the given user account access to SSH into the instance
 * https://cloud.google.com/iap/docs/tutorial-gce 
 */
resource "google_iap_tunnel_instance_iam_binding" "ssh_access" {
  count    = var.allow_ssh ? 1 : 0
  project  = google_compute_instance.vpn.project
  zone     = google_compute_instance.vpn.zone
  instance = google_compute_instance.vpn.name
  role     = "roles/iap.tunnelResourceAccessor"
  members = [
    "user:${var.ssh_user_email}"
  ]
}
