output "omisego_vpc_test_instance_ip" {
  value = google_compute_instance.omisego_test.network_interface.0.network_ip
}

output "omisego_vpc_test_instance_ssh_command" {
  value = "gcloud beta compute ssh --zone ${google_compute_instance.omisego_test.zone} ${google_compute_instance.omisego_test.name} --tunnel-through-iap --project ${var.gcp_project_omisego}"
}
