/*
 * Google Compute Network - https://www.terraform.io/docs/providers/google/r/compute_network.html
 * Defines VPC where Vault infrastructure is provisioned into
 */
resource "google_compute_network" "vpc" {
  name                    = "vault-net"
  auto_create_subnetworks = false
  routing_mode            = "REGIONAL"
}

/*
 * Google Compute Subnetwork - https://www.terraform.io/docs/providers/google/r/compute_subnetwork.html
 * Defines regional subnet where Vault infrastructure is provisioned into
 */
resource "google_compute_subnetwork" "subnet" {
  name          = "vault-subnet"
  ip_cidr_range = var.vault_subnet_cidr
  region        = var.gcp_region
  network       = google_compute_network.vpc.self_link

  # Note: Immutability recommends enabling flow logs for observability, debugging, and incident response.
  # These incur in additional cost.
  log_config {
    aggregation_interval = "INTERVAL_10_MIN"
    flow_sampling        = 1
    metadata             = "INCLUDE_ALL_METADATA"
  }
}

/*
 * Google Compute Router - https://www.terraform.io/docs/providers/google/r/compute_router.html
 * Routes subnet traffic
 */
resource "google_compute_router" "router" {
  name    = "vault-net-router"
  region  = google_compute_subnetwork.subnet.region
  network = google_compute_network.vpc.self_link

  bgp {
    asn = var.router_asn
  }
}

/*
 * Google Compute Route NAT - https://www.terraform.io/docs/providers/google/r/compute_router_nat.html
 * Routes internet bound traffic from instances with no public IPs
 */
resource "google_compute_router_nat" "nat" {
  name                               = "vault-net-router-nat"
  router                             = google_compute_router.router.name
  region                             = google_compute_router.router.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  # Note: Immutability recommends enabling flow logs for observability, debugging, and incident response.
  # These would incur in additional cost.
  log_config {
    enable = true
    filter = "ALL"
  }
}

/*
 * Network Peering - https://www.terraform.io/docs/providers/google/r/compute_network_peering.html
 * Connecting OMGNetwork VPC
 */
resource "google_compute_network_peering" "peering" {
  name         = "peering-to-omgnetwork-vpc"
  network      = google_compute_network.vpc.self_link
  peer_network = var.omgnetwork_vpc_uri
}
