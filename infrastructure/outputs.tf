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