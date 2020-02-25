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
  name    = "datadog-log-egress"
  network = google_compute_network.vpc.name

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

  source_ranges = data.datadog_ip_ranges.ips.logs_ipv4
}

resource "google_compute_firewall" "datadog_agent_1_egress" {
  name    = "datadog-agent-1-egress"
  network = google_compute_network.vpc.name

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

  source_ranges = element(chunklist(data.datadog_ip_ranges.ips.agents_ipv4, 256), 0)
}

resource "google_compute_firewall" "datadog_agent_2_egress" {
  name    = "datadog-agent-2-egress"
  network = google_compute_network.vpc.name

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

  source_ranges = element(chunklist(data.datadog_ip_ranges.ips.agents_ipv4, 256), 1)
}

resource "google_compute_firewall" "omisego_vpc_access" {
  name        = "omisego-vpc-access"
  network     = google_compute_network.vpc.name
  description = "Allows access from Omisego VPC"

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = ["8200"]
  }

  source_ranges = [var.omisego_subnet_cidr]
  target_tags   = ["vault"]
}