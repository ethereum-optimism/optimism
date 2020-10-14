output "vpc_id" {
  value = google_compute_network.vpc.id
}

output "vpc_uri" {
  value = google_compute_network.vpc.self_link
}
