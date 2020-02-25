output "vault_vpc_test_instance_ip" {
  value = google_compute_instance.nat_test.network_interface.0.network_ip
}

output "vault_vpc_test_instance_ssh_command" {
  value = "gcloud beta compute ssh ${google_compute_instance.nat_test.name} --tunnel-through-iap"
}

output "omisego_vpc_test_instance_ip" {
  value = google_compute_instance.omisego_test.network_interface.0.network_ip
}

output "omisego_vpc_test_instance_ssh_command" {
  value = "gcloud beta compute ssh ${google_compute_instance.omisego_test.name} --tunnel-through-iap"
}
