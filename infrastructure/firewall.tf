/*
 * Data Dog IP Ranges - https://www.terraform.io/docs/providers/datadog/d/ip_ranges.html
 * This data source provides the IP ranges required to direct traffic to Data Dog for
 * logging and monitoring purposes.
 */
data "datadog_ip_ranges" "ips" {}

/*
 * Google Compute Firewall - https://www.terraform.io/docs/providers/google/r/compute_firewall.html
 * Egrees rule that allows traffic to be directed to Data Dog's network
 */
resource "google_compute_firewall" "datadog_egress" {
  name    = "datadog-egress"
  network = google_compute_network.vpc.name

  # Allowed ports are configured for Data Dog following requirements specified here: 
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

  source_ranges = concat(
    data.datadog_ip_ranges.ips.agents_ipv4,
    data.datadog_ip_ranges.ips.logs_ipv4,
    data.datadog_ip_ranges.ips.agents_ipv6,
    data.datadog_ip_ranges.ips.logs_ipv6
  )
}