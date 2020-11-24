/*
 * Google DNS Managed Zone: https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_managed_zone
 * Peers the Vault Kubernetes service DNS into the OMGNetwork VPC
 */
resource "google_dns_managed_zone" "vault" {
  name        = "vault-service-dns-zone"
  dns_name    = var.vault_dns
  description = "Vault service DNS zone to be peered into the OMGNetwork VPC"
  visibility  = "private"

  private_visibility_config {
    networks {
      network_url = google_compute_network.vpc.id
    }
  }

  peering_config {
    target_network {
      network_url = var.omgnetwork_vpc_uri
    }
  }
}
