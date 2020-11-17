output "vpc_id" {
  value       = google_compute_network.vpc.id
  description = "The identifier of the VPC."
}

output "vpc_uri" {
  value       = google_compute_network.vpc.self_link
  description = "URI of the VPC."
}

output "subnet_uri" {
  value       = google_compute_subnetwork.subnet.self_link
  description = "URI of the Vault subnet"
}

output "registry_uri" {
  value       = google_container_registry.registry.bucket_self_link
  description = "The self-link URI for the private container registry in GCR"
}

output "gke_services_cidr" {
  value       = google_container_cluster.cluster.services_ipv4_cidr
  description = "CIDR designated for Kubernetes service endpoints"
}

output "dns_managed_zone_id" {
  value = google_dns_managed_zone.vault.id
  description = "DNS Managed Zone ID"
}

output "dns_managed_zone_name_servers" {
  value = google_dns_managed_zone.vault.name_servers
  description = "DNS Managed Zone Name Servers"
}
