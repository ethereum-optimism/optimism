output "vault_vpc_test_instance_ip" {
  value = google_compute_instance.test.network_interface.0.network_ip
}

output "vault_vpc_test_instance_ssh_command" {
  value = "gcloud beta compute ssh --zone ${google_compute_instance.test.zone} ${google_compute_instance.test.name} --tunnel-through-iap --project ${var.gcp_project}"
}
