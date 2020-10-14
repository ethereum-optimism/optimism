/*
 * GPC Zones - https://www.terraform.io/docs/providers/google/d/google_compute_zones.html
 * Used to select the zone where the instance is going to be provissioned
 */
data "google_compute_zones" "available" {
  region = var.gcp_region
}

/*
 * IAP Tunnel - https://www.terraform.io/docs/providers/google/r/iap_tunnel_instance_iam.html
 * An IAP tunnel grants the given user account access to SSH into the instance
 * https://cloud.google.com/iap/docs/tutorial-gce 
 */
resource "google_iap_tunnel_instance_iam_binding" "ssh_access" {
  count    = var.allow_ssh ? 1 : 0
  project  = google_compute_instance.vpn.project
  zone     = google_compute_instance.vpn.zone
  instance = google_compute_instance.vpn.name
  role     = "roles/iap.tunnelResourceAccessor"
  members = [
    "user:${var.ssh_user_email}"
  ]
}

/*
 * Service Account - https://www.terraform.io/docs/providers/google/r/google_service_account.html
 * The client VPN configuration is stored in a bucket using the below service account
 */
resource "google_service_account" "vpn" {
  account_id   = "vpnservice"
  display_name = "VPN Service Account"
  description  = "Service account used by VPN instance to store vpn client config in bucket"
}

/*
 * IAM Member - https://www.terraform.io/docs/providers/google/r/google_project_iam.html#google_project_iam_member-1
 * Grants service account access to manage storage objects
 */
resource "google_project_iam_member" "project" {
  project = google_compute_instance.vpn.project
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.vpn.email}"
}

/*
 * Compute Address - https://www.terraform.io/docs/providers/google/r/compute_address.html
 * Static public IP address that's attached to the VPN instance
 */
resource "google_compute_address" "vpn_address" {
  name = "vpn-address"
}

/*
 * Storage Bucket - https://www.terraform.io/docs/providers/google/r/storage_bucket.html
 * Bucket used for storing the OpenVPN client configuration file
 */
resource "google_storage_bucket" "vpn" {
  name          = var.vpn_bucket_name
  location      = "US"
  force_destroy = true
}

/*
 * Compute Instance - https://www.terraform.io/docs/providers/google/r/compute_instance.html
 * Instance running OpenVPN service
 * The metadata script will download the open VPN installation script here: 
 * https://github.com/angristan/openvpn-install/blob/master/openvpn-install.sh
 * and install openvpn generating an OpenVPN configuration file that's would be used 
 * by the user to connect to the network. 
 * For backup purposes, a copy of the OpenVPN install script has been stored in this repository 
 * on the scripts directory.
 * Connections to the OpenVPN instance have been tested using Tunnelblick on mac os https://tunnelblick.net/
 */
resource "google_compute_instance" "vpn" {
  name           = "vpn"
  machine_type   = "f1-micro"
  zone           = data.google_compute_zones.available.names[0] # "us-east4-a"
  can_ip_forward = true

  tags = ["ssh-access", "vpn"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-9"
    }
  }

  service_account {
    email  = google_service_account.vpn.email
    scopes = ["storage-rw"]
  }

  network_interface {
    subnetwork = google_compute_subnetwork.subnet.self_link
    access_config {
      nat_ip = google_compute_address.vpn_address.address
    }
  }

  metadata_startup_script = <<-EOT
    apt-get update -qq
    curl -O https://raw.githubusercontent.com/angristan/openvpn-install/master/openvpn-install.sh
    chmod +x openvpn-install.sh
    AUTO_INSTALL=y \
        APPROVE_IP=${google_compute_address.vpn_address.address} \
        CLIENT=config \
        DNS=3 \
        PASS=1 \
        ./openvpn-install.sh
    gsutil cp /root/config.ovpn ${google_storage_bucket.vpn.url}
    EOT
}

/*
 * Compute Route - https://www.terraform.io/docs/providers/google/r/compute_route.html
 * Routes traffic that originates from Vault instances in the VPC to the unselear Vault through the VPN
 * https://openvpn.net/vpn-server-resources/google-cloud-platform-byol-instance-quick-start-guide/
 * https://community.openvpn.net/openvpn/wiki/BridgingAndRouting
 */
resource "google_compute_route" "vpn-outbound" {
  name                   = "vpn-outbound"
  dest_range             = "10.8.0.0/24"
  network                = google_compute_network.vpc.name
  tags                   = ["vault"]
  next_hop_instance      = google_compute_instance.vpn.self_link
  next_hop_instance_zone = google_compute_instance.vpn.zone
  priority               = 500
}
