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

output "vpn_private_instance_ip" {
  value       = google_compute_instance.vpn.network_interface.0.network_ip
  description = "Internal IP address of the VPN instance"
}

output "vpn_instance_ssh_command" {
  value       = "gcloud beta compute ssh --zone ${google_compute_instance.vpn.zone} ${google_compute_instance.vpn.name} --tunnel-through-iap --project ${var.gcp_project}"
  description = "SSH command to access VPN instance"
}

output "vpn_public_instance_ip" {
  value       = google_compute_address.vpn_address.address
  description = "Public IP address of the VPN instance"
}

output "bucket_ovpn_command" {
  value       = "gsutil cp gs://${var.bucket_name}/unsealer.ovpn ."
  description = "Command to retrieve ovpn file"
}