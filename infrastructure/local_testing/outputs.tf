output "instance_ip" {
  value = google_compute_instance.nat_test.network_interface.0.network_ip
}

output "instance_ssh_command" {
  value = "gcloud beta compute ssh ${google_compute_instance.nat_test.name} --tunnel-through-iap"
}
