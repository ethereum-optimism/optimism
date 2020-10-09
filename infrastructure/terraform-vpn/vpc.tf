/*
 * Google Compute Network - https://www.terraform.io/docs/providers/google/r/compute_network.html
 * Defines VPC where Vault infrastructure is provisioned into
 */
resource "google_compute_network" "vpc" {
  name                    = "omgnetwork-net"
  auto_create_subnetworks = "false"
  routing_mode            = "REGIONAL"
}

/*
 * Google Compute Subnetwork - https://www.terraform.io/docs/providers/google/r/compute_subnetwork.html
 * Defines regional subnet where Vault infrastructure is provisioned into
 */
resource "google_compute_subnetwork" "subnet" {
  name          = "omgnetwork-subnet"
  ip_cidr_range = var.subnet_cidr
  region        = var.gcp_region
  network       = google_compute_network.vpc.self_link

  # Note: Immutability recommends enabling flow logs for observability, debugging, and incident response.
  # These would incur in additional cost.
  log_config {
    aggregation_interval = "INTERVAL_10_MIN"
    flow_sampling        = 1
    metadata             = "INCLUDE_ALL_METADATA"
  }
}

/*
 * Network Peering - https://www.terraform.io/docs/providers/google/r/compute_network_peering.html
 * Connecting VPC with clients to VPC hosting Vault
 */
resource "google_compute_network_peering" "peering" {
  name         = "peering-to-vault-vpc"
  network      = google_compute_network.vpc.self_link
  peer_network = var.vault_vpc_uri
}

/*
 * This firewall rule allows IAP access to the network for SSH
 * https://www.terraform.io/docs/providers/google/r/compute_firewall.html
 */
resource "google_compute_firewall" "omgnetwork_ssh_iap" {
  name    = "ssh-access"
  network = google_compute_network.vpc.name

  source_ranges = var.ssh_cidr_list

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_tags = ["ssh-access"]
}

/*
 * This grants the given user account access to SSH into the instance
 */
resource "google_iap_tunnel_instance_iam_binding" "omgnetwork_editor" {
  project  = google_compute_instance.omgnetwork_test.project
  zone     = google_compute_instance.omgnetwork_test.zone
  instance = google_compute_instance.omgnetwork_test.name
  role     = "roles/iap.tunnelResourceAccessor"
  members = [
    "user:${var.ssh_user}"
  ]
}

/*
 * Instance used to test connectivity from omgnetwork VPC
 */
resource "google_compute_instance" "omgnetwork_test" {
  name         = "omgnetwork-testing"
  machine_type = "f1-micro"
  zone         = var.gcp_zone

  tags = ["ssh-access"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-9"
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.subnet.self_link
  }
}
