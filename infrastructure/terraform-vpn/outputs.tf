output "omgnetwork_vpc_test_instance_ip" {
  value = google_compute_instance.omgnetwork_test.network_interface.0.network_ip
}

output "omgnetwork_cidr" {
  value = var.subnet_cidr
}

output "vpc_id" {
  value = google_compute_network.vpc.id
}

output "vpc_uri" {
  value = google_compute_network.vpc.self_link
}

output "omgnetwork_vpc_test_instance_ssh_command" {
  value = "gcloud beta compute ssh --zone ${google_compute_instance.omgnetwork_test.zone} ${google_compute_instance.omgnetwork_test.name} --tunnel-through-iap --project ${var.gcp_project}"
}
