/*
 * Google Kubernetes Engine Cluster - https://www.terraform.io/docs/providers/google/r/container_cluster.html
 * Creates the GKE cluster that will be running the Vault and Consul pods
 */
resource "google_container_cluster" "cluster" {
  name     = var.gke_cluster_name
  location = var.gcp_region

  remove_default_node_pool = true
  initial_node_count       = 1

  master_auth {
    username = ""
    password = ""

    client_certificate_config {
      issue_client_certificate = false
    }
  }
}

/*
 * GKE Cluster Node Pool - https://www.terraform.io/docs/providers/google/r/container_node_pool.html
 * Custom node pool definition to allow future control instead of using default
 */
resource "google_container_node_pool" "pool" {
  name       = "${var.gke_cluster_name}-node-pool"
  location   = var.gcp_region
  cluster    = google_container_cluster.cluster.name
  node_count = var.gke_node_count

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  node_config {
    preemptible  = true
    machine_type = "n1-standard-1"

    metadata = {
      disable-legacy-endpoints = true
    }

    oauth_scopes = [
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
    ]
  }
}
