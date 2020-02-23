/*
 * This firewall rule allows IAP access to the network for SSH
 * https://cloud.google.com/nat/docs/gce-example?hl=en_US
 */
resource "google_compute_firewall" "ssh_iap" {
  name    = "ssh-access"
  network = var.vault_vpc_name

  source_ranges = ["35.235.240.0/20"]

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_tags = ["ssh-access"]
}

/*
 * This grants the given user account access to SSH into the instance
 */
resource "google_iap_tunnel_instance_iam_binding" "editor" {
  project  = google_compute_instance.nat_test.project
  zone     = google_compute_instance.nat_test.zone
  instance = google_compute_instance.nat_test.name
  role     = "roles/iap.tunnelResourceAccessor"
  members = [
    "user:${var.ssh_user}"
  ]
}

/*
 * Instance used to test Vault VPC's connectivity
 */
resource "google_compute_instance" "nat_test" {
  name         = "nat-testing"
  machine_type = "f1-micro"
  zone         = "us-central1-a"

  tags = ["ssh-access"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-9"
    }
  }

  network_interface {
    subnetwork = var.vault_vpc_subnet
  }

  metadata_startup_script = <<-EOT
    DD_AGENT_MAJOR_VERSION=7 
    DD_API_KEY=var.datadog_api_key
    bash -c "$(curl -L https://raw.githubusercontent.com/DataDog/datadog-agent/master/cmd/agent/install_script.sh)"
    sudo apt-get -yq install stress
    stress -c 8 -t 120
    EOT
}
