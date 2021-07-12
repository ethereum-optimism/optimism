/*
 * Google Compute Network - https://www.terraform.io/docs/providers/google/r/compute_network.html
 * Defines VPC where Vault infrastructure is provisioned into
 */
resource "google_compute_network" "vpc" {
  name                    = "omgnetwork-net"
  auto_create_subnetworks = true
  routing_mode            = "REGIONAL"
}

/*
 * Network Peering - https://www.terraform.io/docs/providers/google/r/compute_network_peering.html
 * Connecting OMGNetwork VPC
 */
resource "google_compute_network_peering" "peering" {
  name         = "peering-to-omgnetwork-vpc"
  network      = google_compute_network.vpc.self_link
  peer_network = "https://www.googleapis.com/compute/v1/projects/omgnetwork-vault-292019/global/networks/vault-net"
}
