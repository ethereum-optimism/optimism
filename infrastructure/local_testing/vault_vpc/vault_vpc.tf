/*
 * This grants the given user account access to SSH into the instance
 */
resource "google_iap_tunnel_instance_iam_binding" "editor" {
  project  = google_compute_instance.test.project
  zone     = google_compute_instance.test.zone
  instance = google_compute_instance.test.name
  role     = "roles/iap.tunnelResourceAccessor"
  members = [
    "user:${var.ssh_user}"
  ]
}

/*
 * Instance used to test Vault VPC's connectivity
 */
resource "google_compute_instance" "test" {
  name         = "test"
  machine_type = "f1-micro"
  zone         = var.gcp_zone

  tags = [
    "ssh-access",
    "vault"
  ]

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
    apt-get update -qq
    sudo apt-get -yq install stress unzip less
    wget https://releases.hashicorp.com/vault/1.3.2/vault_1.3.2_linux_amd64.zip
    unzip vault_1.3.2_linux_amd64.zip
    mv vault /usr/local/bin/
    vault server -dev -dev-listen-address="0.0.0.0:8200" -dev-root-token-id="totally-secure" -log-level=debug &
    EOT
}
