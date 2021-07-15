/*
 * Datadog IP Ranges - https://www.terraform.io/docs/providers/datadog/d/ip_ranges.html
 * This data source provides the IP ranges required to direct traffic to Datadog for
 * logging and monitoring purposes.
 */
data "datadog_ip_ranges" "ips" {}

/*
 * Google Compute Firewall - https://www.terraform.io/docs/providers/google/r/compute_firewall.html
 * Egrees rule that allows traffic to be directed to Datadog's network
 */
resource "google_compute_firewall" "datadog_logs_egress" {
  name      = "datadog-log-egress"
  network   = google_compute_network.vpc.name
  direction = "EGRESS"
  priority  = "64000"

  # Allowed ports are configured for Datadog following requirements specified here: 
  # https://docs.datadoghq.com/agent/guide/network/?tab=agentv6v7

  allow {
    protocol = "tcp"
    ports = [
      "443",   # port for most Agent data. (Metrics, APM, Live Processes/Containers)
      "10516", # port for the Log collection over TCP
      "10255", # port for the Kubernetes http kubelet
      "10250"  # port for the Kubernetes https kubelet
    ]
  }

  allow {
    protocol = "udp"
    ports = [
      "123" # Used for NTP traffic
    ]
  }

  destination_ranges = data.datadog_ip_ranges.ips.logs_ipv4
}

resource "google_compute_firewall" "datadog_agent_1_egress" {
  name      = "datadog-agent-1-egress"
  network   = google_compute_network.vpc.name
  direction = "EGRESS"
  priority  = "64100"

  # Allowed ports are configured for Datadog following requirements specified here: 
  # https://docs.datadoghq.com/agent/guide/network/?tab=agentv6v7

  allow {
    protocol = "tcp"
    ports = [
      "443",   # port for most Agent data. (Metrics, APM, Live Processes/Containers)
      "10516", # port for the Log collection over TCP
      "10255", # port for the Kubernetes http kubelet
      "10250"  # port for the Kubernetes https kubelet
    ]
  }

  allow {
    protocol = "udp"
    ports = [
      "123" # Used for NTP traffic
    ]
  }

  destination_ranges = element(chunklist(data.datadog_ip_ranges.ips.agents_ipv4, 256), 0)
}

resource "google_compute_firewall" "datadog_agent_2_egress" {
  name      = "datadog-agent-2-egress"
  network   = google_compute_network.vpc.name
  direction = "EGRESS"
  priority  = "64200"

  # Allowed ports are configured for Datadog following requirements specified here: 
  # https://docs.datadoghq.com/agent/guide/network/?tab=agentv6v7

  allow {
    protocol = "tcp"
    ports = [
      "443",   # port for most Agent data. (Metrics, APM, Live Processes/Containers)
      "10516", # port for the Log collection over TCP
      "10255", # port for the Kubernetes http kubelet
      "10250"  # port for the Kubernetes https kubelet
    ]
  }

  allow {
    protocol = "udp"
    ports = [
      "123" # Used for NTP traffic
    ]
  }

  destination_ranges = element(chunklist(data.datadog_ip_ranges.ips.agents_ipv4, 256), 1)
}

resource "google_compute_firewall" "https_egress" {
  name      = "https-egress"
  network   = google_compute_network.vpc.name
  direction = "EGRESS"
  priority  = "64300"

  allow {
    protocol = "tcp"
    ports    = ["443"]
  }

  # Infura and LetsEncrypt egress.
  # Covers all IP addresses as no IP range can be given for these services.
  destination_ranges = ["0.0.0.0/0"]
}

/*
 * This firewall rule contains the required access from the OMGNetwork VPC to access Vault
 */
resource "google_compute_firewall" "omgnetwork_vpc_access" {
  name        = "omgnetwork-vpc-access"
  network     = google_compute_network.vpc.name
  description = "Allows access from OMGNetwork VPC"
  direction   = "INGRESS"
  priority    = "1000"

  # ICMP is allow in order to test connectivity between networks using ping
  allow {
    protocol = "icmp"
  }

  # Port Vault listens in
  allow {
    protocol = "tcp"
    ports    = ["8200"]
  }

  source_ranges = var.omgnetwork_cidrs
}

/*
 * This firewall rule allows IAP access to the network for SSH
 * SSH should only be used in emergency situations
 * https://www.terraform.io/docs/providers/google/r/compute_firewall.html
 */
resource "google_compute_firewall" "ssh_iap" {
  name      = "ssh-iap-access"
  network   = google_compute_network.vpc.name
  direction = "INGRESS"
  priority  = "1100"

  source_ranges = ["35.235.240.0/20"]

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_tags = ["ssh-access"]
}

/*	
 * This firewall rules for VPN ingress access	
 */
resource "google_compute_firewall" "vpn_internet" {
  name          = "vpn-internet"
  network       = google_compute_network.vpc.name
  direction     = "INGRESS"
  priority      = "1200"
  source_ranges = ["0.0.0.0/0"]

  allow {
    protocol = "udp"
    ports    = ["1194"]
  }

  target_tags = ["vpn"]
}

resource "google_compute_firewall" "vpn_outbound" {
  name      = "vpn-access"
  network   = google_compute_network.vpc.name
  direction = "INGRESS"
  priority  = "1300"

  allow {
    protocol = "udp"
  }

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
  }

  source_tags = ["vault"]
  target_tags = ["vpn"]
}
